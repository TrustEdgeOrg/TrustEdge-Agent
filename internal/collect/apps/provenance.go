package apps

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

var readFileFn = os.ReadFile

// ApplyPackageProvenance fills package manager / identifier / version from
// filesystem layout. Does not shell out to brew/npm.
func ApplyPackageProvenance(id *identity.ApplicationIdentity) {
	if id == nil {
		return
	}
	resolved := id.ResolvedPath
	if resolved == "" {
		resolved = id.ExecutablePath
	}
	if resolved == "" {
		resolved = id.Path
	}
	resolved = posixPath(resolved)
	if resolved == "" {
		return
	}

	if applyHomebrewProvenance(id, resolved) {
		return
	}
	applyNpmProvenance(id, resolved)
}

func applyHomebrewProvenance(id *identity.ApplicationIdentity, resolved string) bool {
	// .../Cellar/<formula>/<version>/...
	parts := strings.Split(resolved, "/")
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "Cellar") {
			continue
		}
		formula := parts[i+1]
		version := parts[i+2]
		if formula == "" || version == "" {
			continue
		}
		id.PackageManager = "homebrew"
		id.PackageIdentifier = formula
		id.PackageVersion = version
		if id.Version == "" {
			id.Version = version
		}
		return true
	}
	// .../homebrew/opt/<formula>/... or .../local/opt/<formula>/...
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] != "opt" || i == 0 {
			continue
		}
		prev := parts[i-1]
		if prev != "homebrew" && prev != "local" {
			continue
		}
		formula := parts[i+1]
		if formula == "" {
			continue
		}
		id.PackageManager = "homebrew"
		id.PackageIdentifier = formula
		return true
	}
	return false
}

func applyNpmProvenance(id *identity.ApplicationIdentity, resolved string) {
	dir := path.Dir(resolved)
	for i := 0; i < 8 && dir != "" && dir != "/" && dir != "."; i++ {
		pkgPath := path.Join(dir, "package.json")
		raw, err := readFileFn(pkgPath)
		if err != nil {
			parent := path.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			continue
		}
		var meta struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil || meta.Name == "" {
			return
		}
		id.PackageManager = "npm"
		id.PackageIdentifier = meta.Name
		id.PackageVersion = meta.Version
		if id.Version == "" {
			id.Version = meta.Version
		}
		if id.EntryPoint == "" {
			id.EntryPoint = posixBase(resolved)
		}
		return
	}
}
