package apps

import (
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// attachIDEExtensions discovers VS Code-compatible extensions under known IDE hosts.
func (e *Engine) attachIDEExtensions(byPath map[string]*InventoryEntry) {
	if e == nil || len(e.extProviders) == 0 {
		return
	}
	hosts := hostIDEsFromInventory(byPath)
	for _, host := range hosts {
		for _, prov := range e.extProviders {
			if prov == nil || !prov.Supports(host) {
				continue
			}
			obs, err := prov.DiscoverInstallations(host)
			if err != nil {
				e.logf("ide extension discover %s: %v", host.ProductID, err)
				continue
			}
			for _, o := range obs {
				app := observationToIdentity(o)
				app = enrichExtensionMetadata(e, app, o.InstallPath)
				eptr := new(InventoryEntry)
				*eptr = e.identifyInstalled(app)
				eptr.Installed = true
				eptr.HostIDEProductID = host.ProductID
				eptr.HostIDEPath = host.Path
				eptr.ExtensionProfile = o.Profile
				eptr.ExtensionID = app.PackageIdentifier
				if eptr.Identification.Product == nil {
					// Unknown / non-AI extensions are dropped from inventory.
					continue
				}
				if !hasEvidence(eptr.Identification.Matched, identity.EvidenceHostIDE) {
					eptr.Identification.Matched = append(eptr.Identification.Matched, identity.EvidenceHostIDE)
				}
				home, _ := homeDirFn()
				disabled, known := readDisabledExtensionIDs(home, host.ProductID)
				applyExtensionEnabledState(eptr, disabled, known)
				byPath[pathKey(app.Path)] = eptr
			}
		}
	}
}

func hostIDEsFromInventory(byPath map[string]*InventoryEntry) []HostIDEIdentity {
	seen := make(map[string]struct{})
	var out []HostIDEIdentity
	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		p := eptr.Identification.Product
		if p.Category != identity.ProductCategoryCodeEditor {
			continue
		}
		id := p.ID
		switch id {
		case identity.ProductCursorID, identity.ProductVSCodeID:
		default:
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, HostIDEIdentity{
			ProductID: id,
			Name:      p.Name,
			Path:      eptr.Identity.Path,
			Family:    "vscode_compatible",
		})
	}
	return out
}

func observationToIdentity(o ExtensionInstallObservation) identity.ApplicationIdentity {
	// Folder names are typically publisher.name-version; treat as discovery hint only.
	idHint := folderExtensionIDHint(o.FolderName)
	return identity.ApplicationIdentity{
		Path:              o.InstallPath,
		ResolvedPath:      o.InstallPath,
		Executable:        o.FolderName,
		ExecutablePath:    o.InstallPath,
		PackageManager:    o.PackageManager,
		PackageIdentifier: idHint,
		Version:           folderExtensionVersionHint(o.FolderName),
	}
}

// folderExtensionIDHint extracts publisher.name from publisher.name-1.2.3 folders.
// Not strong identity — package.json publisher+name is authoritative.
func folderExtensionIDHint(folder string) string {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return ""
	}
	// Strip trailing -version (semver-ish).
	parts := strings.Split(folder, "-")
	if len(parts) < 2 {
		return folder
	}
	// Find last segment that looks like a version start.
	for i := len(parts) - 1; i >= 1; i-- {
		if len(parts[i]) > 0 && (parts[i][0] >= '0' && parts[i][0] <= '9') {
			return strings.Join(parts[:i], "-")
		}
	}
	return folder
}

func folderExtensionVersionHint(folder string) string {
	id := folderExtensionIDHint(folder)
	if id == "" || !strings.HasPrefix(folder, id) {
		return ""
	}
	rest := strings.TrimPrefix(folder, id)
	return strings.TrimPrefix(rest, "-")
}

// enrichExtensionMetadata is filled in by metadata extraction (package.json).
// Default no-op until metadata module is linked; kept as hook for Engine.
var enrichExtensionMetadata = func(e *Engine, app identity.ApplicationIdentity, installPath string) identity.ApplicationIdentity {
	return app
}
