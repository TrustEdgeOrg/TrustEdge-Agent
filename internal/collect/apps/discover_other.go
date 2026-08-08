//go:build !darwin

package apps

import (
	"log"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

type emptyDiscoverer struct{}

func newPlatformDiscoverer(logger *log.Logger) Discoverer {
	_ = logger
	return emptyDiscoverer{}
}

func (emptyDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	return nil, nil
}
