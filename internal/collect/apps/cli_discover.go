package apps

import (
	"log"
	"os"
	"path/filepath"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// CLIDiscoverer finds known AI CLI executables in bounded bin directories.
type CLIDiscoverer struct {
	Log     *log.Logger
	Catalog *identity.Catalog
	RootsFn func() []string
}

func defaultCLIRoots() []string {
	roots := []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/usr/bin",
	}
	if home, err := homeDirFn(); err == nil && home != "" {
		roots = append(roots,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "bin"),
		)
	}
	return roots
}

// Discover returns installed CLI candidates matching the known-AI catalog.
func (d *CLIDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	if d == nil {
		return nil, nil
	}
	catalog := d.Catalog
	if catalog == nil {
		catalog = identity.DefaultCatalog()
	}
	names := CatalogCLINames(catalog)
	if len(names) == 0 {
		return nil, nil
	}
	rootsFn := d.RootsFn
	if rootsFn == nil {
		rootsFn = defaultCLIRoots
	}

	var out []identity.ApplicationIdentity
	seen := make(map[string]struct{})
	for _, root := range rootsFn() {
		root = posixPath(root)
		if root == "" {
			continue
		}
		if _, err := statPathFn(root); err != nil {
			continue
		}
		for name := range names {
			inv := posixPath(filepath.Join(root, name))
			fi, err := lstatFn(inv)
			if err != nil {
				continue
			}
			// Skip directories; accept regular files and symlinks.
			if fi.IsDir() {
				continue
			}
			resolved, err := ResolveExecutable(inv)
			if err != nil || resolved.ResolvedPath == "" {
				d.logf("cli discover: skip %s: %v", inv, err)
				continue
			}
			rfi, err := statPathFn(resolved.ResolvedPath)
			if err != nil || rfi.IsDir() {
				continue
			}
			key := pathKey(resolved.ResolvedPath)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			kind, interp := DetectExecutableKind(resolved.ResolvedPath)
			id := identity.ApplicationIdentity{
				Path:           resolved.ResolvedPath,
				Executable:     name,
				ExecutablePath: resolved.ResolvedPath,
				InvocationPath: resolved.InvocationPath,
				ResolvedPath:   resolved.ResolvedPath,
			}
			if kind == ExecutableKindScript {
				id.Interpreter = interp
				id.EntryPoint = posixBase(resolved.ResolvedPath)
			}
			ApplyPackageProvenance(&id)
			out = append(out, id)
		}
	}
	return out, nil
}

func (d *CLIDiscoverer) logf(format string, args ...any) {
	if d != nil && d.Log != nil {
		d.Log.Printf(format, args...)
	}
}

// Ensure file mode bits are consulted for executability when available.
var _ = os.ModePerm
