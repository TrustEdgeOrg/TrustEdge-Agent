//go:build !darwin && !windows

package collect

import "log"

func newPlatformSecurityWatcher(logger *log.Logger) SecurityWatcher {
	_ = logger
	return nil
}
