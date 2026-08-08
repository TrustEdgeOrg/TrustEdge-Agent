package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestParseExtensionPackageJSON(t *testing.T) {
	raw := []byte(`{
  "name": "claude-dev",
  "publisher": "saoudrizwan",
  "version": "3.8.0",
  "displayName": "Cline"
}`)
	meta, err := parseExtensionPackageJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ExtensionID != "saoudrizwan.claude-dev" {
		t.Fatalf("id=%q", meta.ExtensionID)
	}
	if meta.DisplayName != "Cline" || meta.Version != "3.8.0" {
		t.Fatalf("%+v", meta)
	}
}

func TestReadExtensionPackageJSONMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadExtensionPackageJSON(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestReadExtensionPackageJSONFromDisk(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkg, []byte(`{"name":"continue","publisher":"Continue","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	meta, err := ReadExtensionPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ExtensionID != "Continue.continue" {
		t.Fatalf("id=%q", meta.ExtensionID)
	}
}

func TestExtensionMetaCacheInvalidatesOnChange(t *testing.T) {
	c := newExtensionMetaCache(2)
	fp1 := identity.CacheKey{Path: "/a/package.json", ModTimeUnixNano: 1, Size: 10}
	c.put("/a/package.json", fp1, ExtensionPackageMeta{ExtensionID: "a.b"}, true)
	if meta, ok, hit := c.get("/a/package.json", fp1); !hit || !ok || meta.ExtensionID != "a.b" {
		t.Fatalf("miss %+v %v %v", meta, ok, hit)
	}
	fp2 := identity.CacheKey{Path: "/a/package.json", ModTimeUnixNano: 2, Size: 10}
	if _, _, hit := c.get("/a/package.json", fp2); hit {
		t.Fatal("mtime change must miss")
	}
}

func TestDisplayNameWithoutPublisherNoID(t *testing.T) {
	meta, err := parseExtensionPackageJSON([]byte(`{"displayName":"Cline","name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if meta.ExtensionID != "" {
		t.Fatalf("expected empty id, got %q", meta.ExtensionID)
	}
}
