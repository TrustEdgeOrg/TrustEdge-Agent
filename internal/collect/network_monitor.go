package collect

import (
	"context"
	"log"
	"sync"
	"time"
)

type NetworkChangeReason string

const (
	NetworkReasonInitial   NetworkChangeReason = "initial"
	NetworkReasonLink      NetworkChangeReason = "link_change"
	NetworkReasonHeartbeat NetworkChangeReason = "heartbeat"
)

type NetworkChange struct {
	Reason NetworkChangeReason
}

type NetworkMonitorConfig struct {
	Debounce          time.Duration
	HeartbeatInterval time.Duration
	Logger            *log.Logger
	SummaryPayload    func() map[string]any
}

type NetworkMonitor struct {
	cfg NetworkMonitorConfig
}

func NewNetworkMonitor(cfg NetworkMonitorConfig) *NetworkMonitor {
	if cfg.Debounce <= 0 {
		cfg.Debounce = 2 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 60 * time.Second
	}
	return &NetworkMonitor{cfg: cfg}
}

func (m *NetworkMonitor) Run(ctx context.Context) <-chan NetworkChange {
	out := make(chan NetworkChange, 4)
	signal := make(chan NetworkChangeReason, 8)

	go m.runPlatformWatcher(ctx, signal)

	go func() {
		defer close(out)

		var (
			mu              sync.Mutex
			lastLink        string
			lastPosted      string
			debounceTimer   *time.Timer
			debouncePending NetworkChangeReason
		)

		stopDebounce := func() {
			if debounceTimer != nil {
				debounceTimer.Stop()
				debounceTimer = nil
			}
		}
		defer stopDebounce()

		emit := func(reason NetworkChangeReason) {
			summary := m.cfg.SummaryPayload
			if summary == nil {
				summary = func() map[string]any { return NetworkSummaryPayload() }
			}
			payload := summary()
			fp := summaryFingerprint(payload)
			mu.Lock()
			defer mu.Unlock()
			// Heartbeat always posts so liveness continues when posture is unchanged.
			if reason != NetworkReasonHeartbeat && fp == lastPosted {
				m.logf("network %s: unchanged, skip", reason)
				return
			}
			lastPosted = fp
			lastLink = linkFingerprint()
			select {
			case out <- NetworkChange{Reason: reason}:
				m.logf("network %s", reason)
			case <-ctx.Done():
			}
		}

		scheduleEmit := func(reason NetworkChangeReason) {
			mu.Lock()
			debouncePending = reason
			if debounceTimer == nil {
				debounceTimer = time.AfterFunc(m.cfg.Debounce, func() {
					mu.Lock()
					reason := debouncePending
					debounceTimer = nil
					mu.Unlock()
					emit(reason)
				})
			}
			mu.Unlock()
		}

		checkLink := func(reason NetworkChangeReason) {
			fp := linkFingerprint()
			mu.Lock()
			changed := fp != lastLink
			mu.Unlock()
			if !changed {
				return
			}
			scheduleEmit(reason)
		}

		emit(NetworkReasonInitial)

		heartbeat := time.NewTicker(m.cfg.HeartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case reason := <-signal:
				checkLink(reason)
			case <-heartbeat.C:
				emit(NetworkReasonHeartbeat)
			}
		}
	}()

	return out
}

func (m *NetworkMonitor) logf(format string, args ...any) {
	if m.cfg.Logger != nil {
		m.cfg.Logger.Printf(format, args...)
	}
}
