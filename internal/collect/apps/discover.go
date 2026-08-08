package apps

import (
	"log"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// Discoverer inventories installed applications. Implementations are OS-specific.
type Discoverer interface {
	Discover() ([]identity.ApplicationIdentity, error)
}

// NewDiscoverer returns a platform discoverer. On non-macOS it returns an empty inventory.
func NewDiscoverer(logger *log.Logger) Discoverer {
	return newPlatformDiscoverer(logger)
}
