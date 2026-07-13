//go:build !darwin && !linux && !windows

package collect

func foregroundApp() *ForegroundInfo {
	return nil
}

func idleSeconds() float64 {
	return 0
}
