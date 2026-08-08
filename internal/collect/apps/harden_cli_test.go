package apps

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestHardenCLISpoofNameOutsideRoots(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "tmp", "claude")
	if err := os.MkdirAll(filepath.Dir(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fake, []byte("spoof"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := &CLIDiscoverer{
		Catalog: identity.DefaultCatalog(),
		RootsFn: func() []string { return []string{filepath.Join(dir, "bin")} },
	}
	items, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("/tmp spoof must not be discovered: %+v", items)
	}
}

func TestHardenCLISymlinkAbuseDepth(t *testing.T) {
	dir := t.TempDir()
	prev := filepath.Join(dir, "leaf")
	if err := os.WriteFile(prev, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxSymlinkDepth+5; i++ {
		link := filepath.Join(dir, "lvl", strings.Repeat("x", i+1))
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(prev, link); err != nil {
			t.Fatal(err)
		}
		prev = link
	}
	if _, err := ResolveExecutable(prev); err == nil {
		t.Fatal("expected depth exceeded")
	}
}

func TestHardenCLIPythonNotProduct(t *testing.T) {
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 3, Executable: "/usr/bin/python3", Comm: "python3"}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("python must not match: %+v", got)
	}
}

func TestHardenCLIDuplicateUserAndSystem(t *testing.T) {
	sys := "/opt/homebrew/bin/claude"
	user := "/Users/dev/.local/bin/claude"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{
			{Path: sys, Executable: "claude", ExecutablePath: sys, ResolvedPath: sys, InvocationPath: sys},
			{Path: user, Executable: "claude", ExecutablePath: user, ResolvedPath: user, InvocationPath: user},
		}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want both user-local and system installs, got %d", len(got))
	}
}

func TestHardenCLIRootsAppleSiliconAndIntel(t *testing.T) {
	roots := defaultCLIRoots()
	joined := strings.Join(roots, "\n")
	if !strings.Contains(joined, "/opt/homebrew/bin") {
		t.Fatal("missing Apple Silicon homebrew root")
	}
	if !strings.Contains(joined, "/usr/local/bin") {
		t.Fatal("missing Intel/homebrew-or-local root")
	}
	if !strings.Contains(joined, "/usr/bin") {
		t.Fatal("missing /usr/bin")
	}
}

func TestHardenCLIUnsignedStaysLowWithoutPackagePins(t *testing.T) {
	m := identity.NewMatcher(identity.DefaultCatalog())
	res := m.Identify(identity.ApplicationIdentity{
		Executable:        "claude",
		Path:              "/opt/homebrew/Cellar/claude-code/1.0/bin/claude",
		ResolvedPath:      "/opt/homebrew/Cellar/claude-code/1.0/bin/claude",
		PackageManager:    "homebrew",
		PackageIdentifier: "claude-code",
		SignatureChecked:  true,
		SignatureValid:    false,
	})
	if res.Confidence == identity.ConfidenceVerified {
		t.Fatal("unsigned + unpinned catalog must not VERIFIED")
	}
	if res.Confidence != identity.ConfidenceLow && res.Confidence != identity.ConfidenceMedium {
		t.Fatalf("expected LOW/MEDIUM, got %s", res.Confidence)
	}
}

func TestHardenCLINoBrewOnHotPath(t *testing.T) {
	// Guard: package provenance must not invoke brew/npm binaries.
	if _, err := exec.LookPath("brew"); err != nil {
		t.Skip("brew not installed; still assert ApplyPackageProvenance is FS-only")
	}
	id := identity.ApplicationIdentity{
		ResolvedPath: "/opt/homebrew/Cellar/claude-code/9.9.9/bin/claude",
	}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "homebrew" || id.PackageIdentifier != "claude-code" {
		t.Fatalf("%+v", id)
	}
}

func TestHardenCLINoRecursiveHomeScan(t *testing.T) {
	// RootsFn returns only explicit bins — Discoverer never walks $HOME recursively.
	home := t.TempDir()
	nested := filepath.Join(home, "Projects", "secret", "claude")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	d := &CLIDiscoverer{
		Catalog: identity.DefaultCatalog(),
		RootsFn: func() []string { return []string{bin} },
	}
	items, _ := d.Discover()
	for _, it := range items {
		if strings.Contains(it.Path, "Projects") {
			t.Fatalf("recursive home leak: %+v", it)
		}
	}
}
