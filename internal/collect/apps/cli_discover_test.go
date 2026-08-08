package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestResolveExecutableSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Cellar", "tool", "1.0", "bin", "tool")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "bin", "tool")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveExecutable(link)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsSymlink {
		t.Fatal("expected symlink")
	}
	if pathKey(got.ResolvedPath) != pathKey(target) {
		t.Fatalf("resolved=%q want %q", got.ResolvedPath, target)
	}
}

func TestResolveExecutableLoop(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExecutable(a); err == nil {
		t.Fatal("expected loop error")
	}
}

func TestResolveExecutableBroken(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "missing")
	if err := os.Symlink(filepath.Join(dir, "nope"), link); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExecutable(link); err == nil {
		t.Fatal("expected broken symlink error")
	}
}

func TestCLIDiscovererBoundedRoots(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(bin, "claude")
	if err := os.WriteFile(claude, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Outside roots — must not be discovered.
	tmpFake := filepath.Join(dir, "tmp", "claude")
	if err := os.MkdirAll(filepath.Dir(tmpFake), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFake, []byte("evil"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &CLIDiscoverer{
		Catalog: identity.DefaultCatalog(),
		RootsFn: func() []string { return []string{bin, filepath.Join(dir, "missing-root")} },
	}
	items, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Executable != "claude" {
		t.Fatalf("Executable=%q", items[0].Executable)
	}
	if items[0].Interpreter != "node" {
		t.Fatalf("Interpreter=%q", items[0].Interpreter)
	}
}

func TestDetectExecutableKindShebang(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if err := os.WriteFile(p, []byte("#!/usr/bin/env python3\nprint(1)\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	kind, interp := DetectExecutableKind(p)
	if kind != ExecutableKindScript || interp != "python3" {
		t.Fatalf("kind=%s interp=%s", kind, interp)
	}
}

func TestCompositeDiscovererDedup(t *testing.T) {
	a := stubDiscoverer{apps: []identity.ApplicationIdentity{{Path: "/a", Executable: "a"}}}
	b := stubDiscoverer{apps: []identity.ApplicationIdentity{{Path: "/a", Executable: "a"}, {Path: "/b", Executable: "b"}}}
	got, err := NewCompositeDiscoverer(a, b).Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}
