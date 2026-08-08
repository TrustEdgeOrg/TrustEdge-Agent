package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// VSCodeCompatibleExtensionProvider inventories extensions for VS Code and Cursor.
// Roots are explicit per host — never a recursive home scan.
type VSCodeCompatibleExtensionProvider struct {
	HomeFn func() (string, error)
	// RootsByProduct overrides default extension directories in tests.
	RootsByProduct map[string][]string
}

func (p *VSCodeCompatibleExtensionProvider) home() string {
	fn := p.HomeFn
	if fn == nil {
		fn = homeDirFn
	}
	h, err := fn()
	if err != nil {
		return ""
	}
	return h
}

// Supports reports whether host is a VS Code-compatible IDE we know how to scan.
func (p *VSCodeCompatibleExtensionProvider) Supports(host HostIDEIdentity) bool {
	switch strings.ToLower(strings.TrimSpace(host.ProductID)) {
	case identity.ProductCursorID, identity.ProductVSCodeID:
		return true
	case "":
		return host.Family == "vscode_compatible"
	default:
		return host.Family == "vscode_compatible"
	}
}

func (p *VSCodeCompatibleExtensionProvider) DiscoverInstallations(host HostIDEIdentity) ([]ExtensionInstallObservation, error) {
	if !p.Supports(host) {
		return nil, nil
	}
	roots := p.extensionRoots(host)
	pkgMgr := identity.PackageManagerVSCodeExtension
	if strings.EqualFold(host.ProductID, identity.ProductCursorID) {
		pkgMgr = identity.PackageManagerCursorExtension
	}

	var out []ExtensionInstallObservation
	seen := make(map[string]struct{})
	for _, root := range roots {
		root = posixPath(root)
		if root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}
		profile := profileFromRoot(root)
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			// Skip VS Code/Cursor internal marker dirs.
			if name == "node_modules" {
				continue
			}
			path := posixPath(filepath.Join(root, name))
			key := pathKey(path)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, ExtensionInstallObservation{
				Host:           host,
				InstallPath:    path,
				Profile:        profile,
				FolderName:     name,
				PackageManager: pkgMgr,
			})
		}
	}
	return out, nil
}

func (p *VSCodeCompatibleExtensionProvider) extensionRoots(host HostIDEIdentity) []string {
	if p != nil && p.RootsByProduct != nil {
		if roots, ok := p.RootsByProduct[host.ProductID]; ok {
			return roots
		}
	}
	home := p.home()
	if home == "" {
		return nil
	}
	switch strings.ToLower(host.ProductID) {
	case identity.ProductCursorID:
		return []string{
			filepath.Join(home, ".cursor", "extensions"),
		}
	case identity.ProductVSCodeID:
		return []string{
			filepath.Join(home, ".vscode", "extensions"),
		}
	default:
		return nil
	}
}

func profileFromRoot(root string) string {
	// .../User/profiles/<profile>/extensions → profile name
	parts := strings.Split(posixPath(root), "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "profiles" && parts[i+2] == "extensions" {
			return parts[i+1]
		}
	}
	return ""
}
