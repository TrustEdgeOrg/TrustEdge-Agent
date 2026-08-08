package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestCLIAuxCacheResolveHitMiss(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "claude-real")
	link := filepath.Join(dir, "claude")
	if err := os.WriteFile(target, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	cache := newCLIAuxCache(8)
	r1, err := ResolveExecutableCached(cache, link)
	if err != nil {
		t.Fatal(err)
	}
	if cache.Len() != 1 {
		t.Fatalf("Len=%d", cache.Len())
	}
	r2, err := ResolveExecutableCached(cache, link)
	if err != nil {
		t.Fatal(err)
	}
	if r1.ResolvedPath != r2.ResolvedPath {
		t.Fatalf("%q vs %q", r1.ResolvedPath, r2.ResolvedPath)
	}
	if cache.Len() != 1 {
		t.Fatalf("hit should not grow Len=%d", cache.Len())
	}
}

func TestCLIAuxCacheProvenanceFingerprintChange(t *testing.T) {
	dir := t.TempDir()
	cellar := filepath.Join(dir, "Cellar", "claude-code", "1.0.0", "bin")
	if err := os.MkdirAll(cellar, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(cellar, "claude")
	if err := os.WriteFile(bin, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	cache := newCLIAuxCache(8)
	id := identity.ApplicationIdentity{ResolvedPath: bin}
	ApplyPackageProvenanceCached(cache, &id)
	if id.PackageIdentifier != "claude-code" || id.PackageVersion != "1.0.0" {
		t.Fatalf("%+v", id)
	}
	// Rewrite file → new fingerprint → miss and recompute (same path layout).
	if err := os.WriteFile(bin, []byte("v1-updated-content"), 0o755); err != nil {
		t.Fatal(err)
	}
	id2 := identity.ApplicationIdentity{ResolvedPath: bin}
	ApplyPackageProvenanceCached(cache, &id2)
	if id2.PackageIdentifier != "claude-code" {
		t.Fatalf("%+v", id2)
	}
	if cache.Len() != 2 {
		t.Fatalf("package update should add new fingerprint entry, Len=%d", cache.Len())
	}
}

func TestCLIAuxCacheBounded(t *testing.T) {
	cache := newCLIAuxCache(3)
	for i := 0; i < 10; i++ {
		k := identity.FileFingerprint("/opt/bin/x", int64(i), 1)
		cache.putResolve(k, ResolvedExecutable{ResolvedPath: "/r"})
	}
	if cache.Len() > 3 {
		t.Fatalf("Len=%d", cache.Len())
	}
}

func TestCacheKeyPrefersResolvedPath(t *testing.T) {
	dir := t.TempDir()
	resolved := filepath.Join(dir, "real")
	if err := os.WriteFile(resolved, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := identity.ApplicationIdentity{
		Path:         filepath.Join(dir, "link-name"),
		ResolvedPath: resolved,
	}
	key := cacheKeyForApp(app)
	want := posixPath(resolved)
	if key.Path != want {
		t.Fatalf("key.Path=%q want %q", key.Path, want)
	}
}
