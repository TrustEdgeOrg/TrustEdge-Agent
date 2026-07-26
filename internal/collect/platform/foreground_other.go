//go:build !darwin && !linux && !windows

package platform

func foregroundApp() *ForegroundInfo {
	return nil
}

func idleSeconds() float64 {
	return 0
}
