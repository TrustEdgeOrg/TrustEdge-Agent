package security

import (
	"context"
	"log"
	"sync"
	"time"
)

const defaultSecurityWatchDebounce = 250 * time.Millisecond

// SecurityWatcher streams wake signals when host security artifacts may have changed.
// Callers should run SecurityMonitor.Poll() on each signal; poll reconciliation remains
// the source of truth for baselines and missed events.
type SecurityWatcher interface {
	Run(ctx context.Context) <-chan struct{}
}

// NewSecurityWatcher returns the platform security watcher, or nil when unavailable.
func NewSecurityWatcher(logger *log.Logger) SecurityWatcher {
	return newPlatformSecurityWatcher(logger)
}

// debounceWake coalesces bursty filesystem/registry notifications into single wakes.
func debounceWake(ctx context.Context, in <-chan struct{}, debounce time.Duration) <-chan struct{} {
	out := make(chan struct{}, 1)
	if debounce <= 0 {
		debounce = defaultSecurityWatchDebounce
	}
	go func() {
		defer close(out)
		var (
			mu      sync.Mutex
			timer   *time.Timer
			pending bool
		)
		flush := func() {
			mu.Lock()
			pending = false
			timer = nil
			mu.Unlock()
			select {
			case out <- struct{}{}:
			default:
			}
		}
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				mu.Unlock()
				return
			case _, ok := <-in:
				if !ok {
					mu.Lock()
					wasPending := pending
					if timer != nil {
						timer.Stop()
						timer = nil
					}
					pending = false
					mu.Unlock()
					if wasPending {
						select {
						case out <- struct{}{}:
						default:
						}
					}
					return
				}
				mu.Lock()
				pending = true
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, flush)
				mu.Unlock()
			}
		}
	}()
	return out
}
