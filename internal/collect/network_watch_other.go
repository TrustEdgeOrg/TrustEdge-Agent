//go:build !darwin && !linux && !windows

package collect

import "context"

func (m *NetworkMonitor) runPlatformWatcher(ctx context.Context, signal chan<- NetworkChangeReason) {
	m.logf("network watcher: link events unavailable on this platform (heartbeat only)")
}
