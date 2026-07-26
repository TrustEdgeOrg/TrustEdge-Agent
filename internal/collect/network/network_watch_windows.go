//go:build windows

package network

import (
	"context"
	"time"
)

func (m *NetworkMonitor) runPlatformWatcher(ctx context.Context, signal chan<- NetworkChangeReason) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	last := linkFingerprint()
	m.logf("network watcher: interface polling active")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fp := linkFingerprint()
			if fp == last {
				continue
			}
			last = fp
			select {
			case signal <- NetworkReasonLink:
				m.logf("network link_change (poll)")
			default:
			}
		}
	}
}
