package collect

// ForegroundInfo describes the app in the foreground on the endpoint.
type ForegroundInfo struct {
	BundleID string
	Name     string
}

// foregroundApp and idleSeconds are implemented per platform in foreground_* files.
