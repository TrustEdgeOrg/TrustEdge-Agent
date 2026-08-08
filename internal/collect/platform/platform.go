package platform

// PlatformProbe reads OS-specific foreground app and idle state.
type PlatformProbe interface {
	OSVersion() string
	ForegroundApp() *ForegroundInfo
	IdleSeconds() float64
}

// DefaultProbe uses live platform APIs.
type DefaultProbe struct{}

func (DefaultProbe) OSVersion() string        { return osVersion() }
func (DefaultProbe) ForegroundApp() *ForegroundInfo { return foregroundApp() }
func (DefaultProbe) IdleSeconds() float64       { return idleSeconds() }
