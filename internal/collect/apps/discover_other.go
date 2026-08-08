//go:build !darwin

package apps

import (
	"log"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

type emptyDiscoverer struct{}

func newPlatformDiscoverer(logger *log.Logger) Discoverer {
	return NewCompositeDiscoverer(
		emptyDiscoverer{},
		newCLIDiscoverer(logger),
		newDockerDiscoverer(logger),
	)
}

func (emptyDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	return nil, nil
}
