package apps

import (
	"log"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// CompositeDiscoverer merges multiple discoverers (e.g. .app + CLI).
type CompositeDiscoverer struct {
	Parts []Discoverer
}

// Discover concatenates results from all parts, de-duplicating by path key.
func (c *CompositeDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	if c == nil {
		return nil, nil
	}
	var out []identity.ApplicationIdentity
	seen := make(map[string]struct{})
	for _, part := range c.Parts {
		if part == nil {
			continue
		}
		items, err := part.Discover()
		if err != nil {
			return out, err
		}
		for _, item := range items {
			key := pathKey(item.ResolvedPath)
			if key == "" {
				key = pathKey(item.Path)
			}
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

// NewCompositeDiscoverer builds a discoverer from parts (nil parts skipped).
func NewCompositeDiscoverer(parts ...Discoverer) Discoverer {
	clean := make([]Discoverer, 0, len(parts))
	for _, p := range parts {
		if p != nil {
			clean = append(clean, p)
		}
	}
	return &CompositeDiscoverer{Parts: clean}
}

// newCLIDiscoverer is overridable in tests.
var newCLIDiscoverer = func(logger *log.Logger) Discoverer {
	return &CLIDiscoverer{
		Log:     logger,
		Catalog: identity.DefaultCatalog(),
		Cache:   newCLIAuxCache(cliAuxCacheCapacity),
	}
}
