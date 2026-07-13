package credentials

import "log"

const (
	keyringService        = "TrustEdge Agent"
	keyringServiceLegacy  = "TrustTwin"
	keyringAccount        = "device_token"
)

// Store persists device identity and auth credentials on the endpoint.
type Store interface {
	Load() (deviceID, token string, err error)
	Save(deviceID, token string) error
	ClearToken() error
}

// New returns the platform credential store for statePath.
func New(statePath string, logger *log.Logger) Store {
	return newPlatformStore(statePath, logger)
}
