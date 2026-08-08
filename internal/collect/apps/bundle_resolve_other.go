//go:build !darwin

package apps

import "github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"

// ResolveBundle is unavailable off macOS.
func ResolveBundle(appPath string) (identity.ApplicationIdentity, bool) {
	_ = appPath
	return identity.ApplicationIdentity{}, false
}
