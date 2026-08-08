package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestHomebrewCellarProvenance(t *testing.T) {
	id := identity.ApplicationIdentity{
		ResolvedPath: "/opt/homebrew/Cellar/claude-code/1.2.3/bin/claude",
	}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "homebrew" || id.PackageIdentifier != "claude-code" || id.PackageVersion != "1.2.3" {
		t.Fatalf("%+v", id)
	}
}

func TestNpmPackageJSONProvenance(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@scope", "cli")
	bin := filepath.Join(pkgDir, "bin", "claude.js")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@anthropic/claude-code","version":"9.9.9"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	id := identity.ApplicationIdentity{ResolvedPath: bin}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "npm" || id.PackageIdentifier != "@anthropic/claude-code" || id.PackageVersion != "9.9.9" {
		t.Fatalf("%+v", id)
	}
}

func TestProvenanceMissing(t *testing.T) {
	id := identity.ApplicationIdentity{ResolvedPath: "/usr/bin/true"}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "" || id.PackageIdentifier != "" {
		t.Fatalf("%+v", id)
	}
}

func TestHomebrewOptProvenance(t *testing.T) {
	id := identity.ApplicationIdentity{
		ResolvedPath: "/opt/homebrew/opt/codex/bin/codex",
	}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "homebrew" || id.PackageIdentifier != "codex" {
		t.Fatalf("%+v", id)
	}
}

func TestNpmWrongLayoutLeavesEmpty(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	id := identity.ApplicationIdentity{ResolvedPath: bin}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "" {
		t.Fatalf("unexpected provenance %+v", id)
	}
}
