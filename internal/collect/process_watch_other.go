//go:build !darwin && !linux && !windows

package collect

import "log"

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	_ = logger
	return nil
}
