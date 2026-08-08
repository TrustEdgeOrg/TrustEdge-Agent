//go:build darwin && !cgo

package process

import "log"

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	if logger != nil {
		logger.Printf("process watcher: CGO disabled on darwin; using poll reconciliation only")
	}
	return nil
}
