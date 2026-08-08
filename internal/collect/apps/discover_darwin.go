//go:build darwin

package apps

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

var (
	readDirFn = os.ReadDir
	statFn    = os.Stat
	plutilFn  = func(path string) ([]byte, error) {
		return exec.Command("plutil", "-convert", "json", "-o", "-", "--", path).Output()
	}
	applicationRootsFn = defaultApplicationRoots
)

type darwinDiscoverer struct {
	log *log.Logger
}

func newPlatformDiscoverer(logger *log.Logger) Discoverer {
	return NewCompositeDiscoverer(
		&darwinDiscoverer{log: logger},
		newCLIDiscoverer(logger),
		newDockerDiscoverer(logger),
	)
}

func (d *darwinDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	roots := applicationRootsFn()
	var out []identity.ApplicationIdentity
	seen := make(map[string]struct{})

	for _, root := range roots {
		apps, err := scanApplicationsDir(root)
		if err != nil {
			d.logf("apps discover: skip %s: %v", root, err)
			continue
		}
		for _, app := range apps {
			key := strings.ToLower(app.Path)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, app)
		}
	}
	return out, nil
}

func defaultApplicationRoots() []string {
	roots := []string{"/Applications"}
	if home, err := homeDirFn(); err == nil && home != "" {
		roots = append(roots, filepath.Join(home, "Applications"))
	}
	return roots
}

func scanApplicationsDir(root string) ([]identity.ApplicationIdentity, error) {
	entries, err := readDirFn(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []identity.ApplicationIdentity
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".app") {
			continue
		}
		// Prefer Dir entries; also accept bundles that appear as files on some FS views.
		path := filepath.Join(root, name)
		id, ok := readAppBundle(path)
		if !ok {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func readAppBundle(appPath string) (identity.ApplicationIdentity, bool) {
	infoPath := filepath.Join(appPath, "Contents", "Info.plist")
	if _, err := statFn(infoPath); err != nil {
		return identity.ApplicationIdentity{}, false
	}
	meta, err := parseInfoPlist(infoPath)
	if err != nil {
		return identity.ApplicationIdentity{}, false
	}

	id := identity.ApplicationIdentity{
		Path:       appPath,
		BundleID:   meta.BundleID,
		Version:    meta.Version,
		Executable: meta.Executable,
	}
	if meta.Executable != "" {
		id.ExecutablePath = filepath.Join(appPath, "Contents", "MacOS", meta.Executable)
	}
	return id, true
}

type bundleMeta struct {
	BundleID   string
	Executable string
	Version    string
}

func parseInfoPlist(path string) (bundleMeta, error) {
	raw, err := plutilFn(path)
	if err != nil {
		return bundleMeta{}, err
	}
	var plist map[string]any
	if err := json.Unmarshal(raw, &plist); err != nil {
		return bundleMeta{}, err
	}
	meta := bundleMeta{
		BundleID:   stringFromAny(plist["CFBundleIdentifier"]),
		Executable: stringFromAny(plist["CFBundleExecutable"]),
		Version:    firstNonEmpty(stringFromAny(plist["CFBundleShortVersionString"]), stringFromAny(plist["CFBundleVersion"])),
	}
	return meta, nil
}

func stringFromAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func (d *darwinDiscoverer) logf(format string, args ...any) {
	if d.log != nil {
		d.log.Printf(format, args...)
	}
}
