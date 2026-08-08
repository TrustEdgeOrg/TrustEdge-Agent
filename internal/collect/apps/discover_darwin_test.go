//go:build darwin

package apps

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanApplicationsDir(t *testing.T) {
	root := t.TempDir()
	writeFakeApp(t, root, "Good.app", `{
		"CFBundleIdentifier": "com.example.good",
		"CFBundleExecutable": "Good",
		"CFBundleShortVersionString": "1.2.3"
	}`)
	writeFakeApp(t, root, "NoPlist.app", "")
	bad := filepath.Join(root, "Bad.app", "Contents")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "Info.plist"), []byte("not-a-plist"), 0o644); err != nil {
		t.Fatal(err)
	}

	origPlutil := plutilFn
	t.Cleanup(func() { plutilFn = origPlutil })
	plutilFn = func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 && data[0] == '{' {
			return data, nil
		}
		return nil, errors.New("malformed plist")
	}

	apps, err := scanApplicationsDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("got %d apps, want 1 (malformed skipped)", len(apps))
	}
	got := apps[0]
	if got.BundleID != "com.example.good" {
		t.Fatalf("BundleID=%q", got.BundleID)
	}
	if got.Executable != "Good" {
		t.Fatalf("Executable=%q", got.Executable)
	}
	if got.Version != "1.2.3" {
		t.Fatalf("Version=%q", got.Version)
	}
	wantExe := filepath.Join(root, "Good.app", "Contents", "MacOS", "Good")
	if got.ExecutablePath != wantExe {
		t.Fatalf("ExecutablePath=%q want %q", got.ExecutablePath, wantExe)
	}
}

func TestDiscoverUsesApplicationsAndHome(t *testing.T) {
	sysRoot := t.TempDir()
	userApps := t.TempDir()
	writeFakeApp(t, sysRoot, "Sys.app", `{
		"CFBundleIdentifier": "com.example.sys",
		"CFBundleExecutable": "Sys",
		"CFBundleShortVersionString": "1.0"
	}`)
	writeFakeApp(t, userApps, "User.app", `{
		"CFBundleIdentifier": "com.example.user",
		"CFBundleExecutable": "User",
		"CFBundleVersion": "9"
	}`)

	origRoots := applicationRootsFn
	origPlutil := plutilFn
	t.Cleanup(func() {
		applicationRootsFn = origRoots
		plutilFn = origPlutil
	})
	applicationRootsFn = func() []string { return []string{sysRoot, userApps} }
	plutilFn = func(path string) ([]byte, error) { return os.ReadFile(path) }

	d := &darwinDiscoverer{}
	got, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d apps want 2: %+v", len(got), got)
	}
	byID := map[string]bool{}
	for _, a := range got {
		byID[a.BundleID] = true
		if a.BundleID == "com.example.user" && a.Version != "9" {
			t.Fatalf("user Version=%q want CFBundleVersion fallback 9", a.Version)
		}
	}
	if !byID["com.example.sys"] || !byID["com.example.user"] {
		t.Fatalf("missing apps: %+v", byID)
	}
}

func TestMissingApplicationsDir(t *testing.T) {
	apps, err := scanApplicationsDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("want empty, got %d", len(apps))
	}
}

func TestDefaultApplicationRoots(t *testing.T) {
	origHome := homeDirFn
	t.Cleanup(func() { homeDirFn = origHome })
	homeDirFn = func() (string, error) { return "/Users/test", nil }
	roots := defaultApplicationRoots()
	if len(roots) != 2 || roots[0] != "/Applications" || roots[1] != "/Users/test/Applications" {
		t.Fatalf("roots=%v", roots)
	}
}

func writeFakeApp(t *testing.T, root, name, infoJSON string) {
	t.Helper()
	contents := filepath.Join(root, name, "Contents")
	macos := filepath.Join(contents, "MacOS")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	if infoJSON == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(infoJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}
