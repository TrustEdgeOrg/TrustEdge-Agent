//go:build !darwin && !linux && !windows

package platform

import "runtime"

func osVersion() string {
	return runtime.GOOS
}
