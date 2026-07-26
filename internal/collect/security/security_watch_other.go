//go:build !darwin && !windows

package security

import "log"

func newPlatformSecurityWatcher(logger *log.Logger) SecurityWatcher {
	_ = logger
	return nil
}
