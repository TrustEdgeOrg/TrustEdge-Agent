package collect

import (
	"context"
	"log"
)

// ProcessWatcher streams event-driven process_start / process_exit changes.
type ProcessWatcher interface {
	Run(ctx context.Context) <-chan ProcessChange
}

// NewProcessWatcher returns the platform process watcher, or nil when unavailable.
func NewProcessWatcher(logger *log.Logger) ProcessWatcher {
	return newPlatformProcessWatcher(logger)
}
