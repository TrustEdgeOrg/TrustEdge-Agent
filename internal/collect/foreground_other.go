//go:build !darwin

package collect

type ForegroundInfo struct {
	BundleID string
	Name     string
}

func foregroundApp() *ForegroundInfo {
	return nil
}

func idleSeconds() float64 {
	return 0
}
