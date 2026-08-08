//go:build darwin

package apps

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHardenApplicationsAndHomeRoots(t *testing.T) {
	sys := t.TempDir()
	home := t.TempDir()
	writeFakeApp(t, sys, "Cursor.app", cursorPlistJSON())
	writeFakeApp(t, home, "Cursor.app", cursorPlistJSON())

	origRoots := applicationRootsFn
	origPlutil := plutilFn
	t.Cleanup(func() {
		applicationRootsFn = origRoots
		plutilFn = origPlutil
	})
	applicationRootsFn = func() []string { return []string{sys, home} }
	plutilFn = func(path string) ([]byte, error) { return os.ReadFile(path) }

	d := &darwinDiscoverer{}
	apps, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("duplicate installs: got %d", len(apps))
	}
}

func TestHardenMalformedInfoPlist(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "Cursor.app", "Contents")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "Info.plist"), []byte("{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := plutilFn
	t.Cleanup(func() { plutilFn = orig })
	plutilFn = func(path string) ([]byte, error) {
		return nil, errors.New("malformed")
	}
	apps, err := scanApplicationsDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("malformed should be skipped, got %d", len(apps))
	}
}

func TestHardenAppRemovedDuringScan(t *testing.T) {
	root := t.TempDir()
	writeFakeApp(t, root, "Cursor.app", cursorPlistJSON())
	origStat := statFn
	t.Cleanup(func() { statFn = origStat })
	statFn = func(path string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	apps, err := scanApplicationsDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("removed app should yield empty, got %d", len(apps))
	}
}

func cursorPlistJSON() string {
	return `{
		"CFBundleIdentifier": "com.todesktop.230313mzl4w4u92",
		"CFBundleExecutable": "Cursor",
		"CFBundleShortVersionString": "1.0"
	}`
}
