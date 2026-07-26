//go:build !darwin && !linux && !windows

package process

import "log"

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	_ = logger
	return nil
}
