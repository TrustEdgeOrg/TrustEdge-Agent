package agent

import (
	"context"
	"log"
	"time"

	"github.com/TrustEdgeOrg/TrustTwin/internal/api"
	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/collect"
	"github.com/TrustEdgeOrg/TrustTwin/internal/config"
	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
	"github.com/TrustEdgeOrg/TrustTwin/internal/credentials"
)

// Dependencies are injected into the agent at composition root.
type Dependencies struct {
	Config    config.AgentConfig
	Logger    *log.Logger
	Clock     clock.Clock
	Client    api.EventClient
	Creds     credentials.Store
	Collector *collect.Collector
}

// Agent reports telemetry to the TrustTwin API.
type Agent struct {
	cfg       config.AgentConfig
	log       *log.Logger
	clock     clock.Clock
	client    api.EventClient
	creds     credentials.Store
	collector *collect.Collector
	deviceID  string
}

func New(deps Dependencies) *Agent {
	clk := deps.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	return &Agent{
		cfg:       deps.Config,
		log:       deps.Logger,
		clock:     clk,
		client:    deps.Client,
		creds:     deps.Creds,
		collector: deps.Collector,
	}
}

// EnsureRegistered registers the device when no credential token is stored.
func (a *Agent) EnsureRegistered(ctx context.Context) error {
	return a.ensureRegistered(ctx)
}

func (a *Agent) Run(ctx context.Context) error {
	batcher := NewEventBatcher(
		a.clock,
		func() string { return a.deviceID },
		a.postEvents,
		a.log,
		a.cfg.EventBatchSize,
		a.cfg.EventBatchFlush,
	)
	go batcher.Run(ctx)

	enqueue := func(typ string, payload map[string]any) {
		batcher.Enqueue(typ, payload)
	}

	enqueue(constants.TypeClientDetails, a.collector.ClientDetailsPayload())

	go a.loop(ctx, a.cfg.DetailsInterval, func() {
		enqueue(constants.TypeClientDetails, a.collector.ClientDetailsPayload())
	})

	monitor := collect.NewNetworkMonitor(collect.NetworkMonitorConfig{
		Debounce:          a.cfg.NetworkDebounce,
		HeartbeatInterval: a.cfg.NetworkInterval,
		Logger:            a.log,
		SummaryPayload:    a.collector.NetworkSummaryPayload,
	})
	go func() {
		for change := range monitor.Run(ctx) {
			a.log.Printf("network event: %s", change.Reason)
			enqueue(constants.TypeNetworkSummary, a.collector.NetworkSummaryPayload())
		}
	}()

	tracker := a.collector.NewActionTracker(a.cfg.ActionInterval)
	go a.loop(ctx, a.cfg.ActionInterval, func() {
		tracker.Sample()
		enqueue(constants.TypeActionSummary, collect.ActionSummaryPayload(tracker.SnapshotAndReset()))
	})

	if a.cfg.ProcessInterval > 0 {
		procMon := collect.NewProcessMonitor(a.log)
		if watcher := collect.NewProcessWatcher(a.log); watcher != nil {
			a.log.Printf("process watcher: event-driven mode active")
			go func() {
				for change := range watcher.Run(ctx) {
					if procMon.Observe(change) {
						enqueue(change.Type, change.Payload)
					}
				}
			}()
		}
		go a.loop(ctx, a.cfg.ProcessInterval, func() {
			for _, change := range procMon.Poll() {
				enqueue(change.Type, change.Payload)
			}
		})
	}

	a.log.Printf("reporting to %s", a.cfg.APIURL)
	<-ctx.Done()
	a.log.Printf("shutting down")
	return ctx.Err()
}

func (a *Agent) loop(ctx context.Context, every time.Duration, fn func()) {
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
