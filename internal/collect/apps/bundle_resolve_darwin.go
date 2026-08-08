//go:build darwin

package apps

import "github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"

// ResolveBundle reads bundle metadata for an .app path.
func ResolveBundle(appPath string) (identity.ApplicationIdentity, bool) {
	return readAppBundle(appPath)
}
