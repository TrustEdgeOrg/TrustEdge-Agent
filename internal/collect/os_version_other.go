//go:build !darwin

package collect

import "runtime"

func osVersion() string {
	return runtime.GOOS
}
