package agent

import (
	"context"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/api"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/action"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/apps"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/security"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/config"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/credentials"
)

// Dependencies are injected into the agent at composition root.
type Dependencies struct {
	Config    config.AgentConfig
	Logger    *slog.Logger
	Clock     clock.Clock
	Client    api.EventClient
	Creds     credentials.Store
	Collector *collect.Collector
	Metrics   *Metrics
}

// Agent reports telemetry to the TrustEdge Agent API.
type Agent struct {
	cfg       config.AgentConfig
	log       *slog.Logger
	stdLog    *log.Logger
	clock     clock.Clock
	client    api.EventClient
	creds     credentials.Store
	collector *collect.Collector
	metrics   *Metrics

	authMu   sync.Mutex
	deviceID string
}

func New(deps Dependencies) *Agent {
	clk := deps.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	metrics := deps.Metrics
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &Agent{
		cfg:       deps.Config,
		log:       logger,
		stdLog:    slog.NewLogLogger(logger.Handler(), slog.LevelInfo),
		clock:     clk,
		client:    deps.Client,
		creds:     deps.Creds,
		collector: deps.Collector,
		metrics:   metrics,
	}
}

// EnsureRegistered loads stored credentials (if any) and always calls
// /v1/register so Agent-API can upsert the agent into TrustEdge Postgres.
// When a token already exists, a re-register failure is non-fatal.
func (a *Agent) EnsureRegistered(ctx context.Context) error {
	return a.ensureRegistered(ctx)
}

func (a *Agent) Run(ctx context.Context) error {
	batcher, err := NewEventBatcher(
		a.clock,
		a.currentDeviceID,
		a.postEvents,
		a.log,
		BatcherOptions{
			MaxSize:    a.cfg.EventBatchSize,
			FlushEvery: a.cfg.EventBatchFlush,
			QueuePath:  a.cfg.EventQueuePath,
			Capacity:   a.cfg.EventQueueCapacity,
			MaxBackoff: a.cfg.EventRetryMax,
			Metrics:    a.metrics,
		},
	)
	if err != nil {
		return err
	}
	go batcher.Run(ctx)

	enqueue := func(typ string, payload map[string]any) {
		batcher.Enqueue(typ, payload)
	}

	enqueue(constants.TypeClientDetails, a.collector.ClientDetailsPayload())

	go a.loop(ctx, a.cfg.DetailsInterval, func() {
		enqueue(constants.TypeClientDetails, a.collector.ClientDetailsPayload())
	})

	monitor := network.NewNetworkMonitor(network.NetworkMonitorConfig{
		Debounce:          a.cfg.NetworkDebounce,
		HeartbeatInterval: a.cfg.NetworkInterval,
		Logger:            a.stdLog,
		SummaryPayload:    a.collector.NetworkSummaryPayload,
	})
	go func() {
		for change := range monitor.Run(ctx) {
			a.log.Info("network event", "reason", change.Reason)
			enqueue(constants.TypeNetworkSummary, change.Payload)
		}
	}()

	sampleEvery := a.cfg.ActionSampleInterval
	if sampleEvery <= 0 {
		sampleEvery = constants.DefaultActionSampleInterval
	}
	if a.cfg.ActionInterval > 0 && sampleEvery > a.cfg.ActionInterval {
		sampleEvery = a.cfg.ActionInterval
	}
	tracker := a.collector.NewActionTracker(sampleEvery)
	go a.loop(ctx, sampleEvery, tracker.Sample)
	go a.loop(ctx, a.cfg.ActionInterval, func() {
		enqueue(constants.TypeActionSummary, action.ActionSummaryPayload(tracker.SnapshotAndReset()))
	})

	if a.cfg.ProcessInterval > 0 {
		procMon := process.NewProcessMonitor(a.stdLog)
		var aiFeed *apps.RuntimeFeed
		if a.cfg.KnownAIInterval > 0 {
			aiFeed = apps.NewRuntimeFeed(a.stdLog, nil)
			go aiFeed.Run(ctx)
		}
		if watcher := process.NewProcessWatcher(a.stdLog); watcher != nil {
			a.log.Info("process watcher active", "mode", "event-driven")
			go func() {
				for change := range watcher.Run(ctx) {
					if aiFeed != nil {
						aiFeed.ObserveChange(change)
					}
					if procMon.Observe(change) {
						enqueue(change.Type, change.Payload)
					}
				}
			}()
		}
		go a.loop(ctx, a.cfg.ProcessInterval, func() {
			for _, change := range procMon.Poll() {
				if aiFeed != nil {
					aiFeed.ObserveChange(change)
				}
				enqueue(change.Type, change.Payload)
			}
		})

		if a.cfg.KnownAIInterval > 0 {
			engine := apps.NewEngine(apps.EngineConfig{Logger: a.stdLog})
			aiMon := apps.NewMonitor(a.stdLog, engine)
			a.log.Info("known-ai inventory active", "interval", a.cfg.KnownAIInterval.String())
			go a.loop(ctx, a.cfg.KnownAIInterval, func() {
				for _, change := range aiMon.Poll() {
					enqueue(change.Type, change.Payload)
				}
			})
			if aiFeed != nil {
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case <-aiFeed.Wakes():
							for _, change := range aiMon.Poll() {
								enqueue(change.Type, change.Payload)
							}
						}
					}
				}()
			}
		}
	} else if a.cfg.KnownAIInterval > 0 {
		aiMon := apps.NewMonitor(a.stdLog, apps.NewEngine(apps.EngineConfig{Logger: a.stdLog}))
		a.log.Info("known-ai inventory active", "interval", a.cfg.KnownAIInterval.String())
		go a.loop(ctx, a.cfg.KnownAIInterval, func() {
			for _, change := range aiMon.Poll() {
				enqueue(change.Type, change.Payload)
			}
		})
	}

	if a.cfg.SecurityInterval > 0 {
		secMon := security.NewSecurityMonitor(a.stdLog)
		if watcher := security.NewSecurityWatcher(a.stdLog); watcher != nil {
			a.log.Info("security watcher active", "mode", "event-driven")
			go func() {
				for range watcher.Run(ctx) {
					for _, change := range secMon.Poll() {
						enqueue(change.Type, change.Payload)
					}
				}
			}()
		}
		go a.loop(ctx, a.cfg.SecurityInterval, func() {
			for _, change := range secMon.Poll() {
				enqueue(change.Type, change.Payload)
			}
		})
	}

	if a.cfg.ConnectionInterval > 0 {
		connMon := network.NewConnectionMonitor(a.stdLog)
		a.log.Info("connection monitor active", "interval", a.cfg.ConnectionInterval.String())
		go a.loop(ctx, a.cfg.ConnectionInterval, func() {
			for _, conn := range connMon.Poll() {
				enqueue(constants.TypeNetworkConnection, conn.Payload())
			}
		})
	}

	if a.cfg.MetricsInterval > 0 {
		go a.loop(ctx, a.cfg.MetricsInterval, func() {
			a.logStatus(batcher)
		})
	}

	a.log.Info("reporting telemetry", "api_url", a.cfg.APIURL, "device_id", a.currentDeviceID())
	<-ctx.Done()
	a.log.Info("shutting down", "device_id", a.currentDeviceID())
	return ctx.Err()
}

func (a *Agent) loop(ctx context.Context, every time.Duration, fn func()) {
	if every <= 0 {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}
