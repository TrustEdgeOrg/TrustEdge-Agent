package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestCatalogExecutableNamesIncludesRuntimes(t *testing.T) {
	names := CatalogExecutableNames(identity.DefaultCatalog())
	for _, n := range []string{"ollama", "llama-server", "llama-cli", "claude"} {
		if _, ok := names[n]; !ok {
			t.Fatalf("missing %s", n)
		}
	}
}

func TestRuntimeDiscovererBoundedRoots(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	ollama := filepath.Join(bin, "ollama")
	if err := os.WriteFile(ollama, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "tmp", "ollama")
	if err := os.MkdirAll(filepath.Dir(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fake, []byte("evil"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &CLIDiscoverer{
		Catalog: identity.DefaultCatalog(),
		RootsFn: func() []string { return []string{bin} },
	}
	items, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range items {
		if it.Executable == "ollama" {
			found = true
		}
		if pathKey(it.Path) == pathKey(fake) {
			t.Fatal("/tmp spoof must not be discovered")
		}
	}
	if !found {
		t.Fatalf("expected ollama in roots, got %+v", items)
	}
}

func TestEngineIdentifiesInstalledOllama(t *testing.T) {
	path := "/opt/homebrew/Cellar/ollama/0.1.0/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
			PackageManager: "homebrew", PackageIdentifier: "ollama", PackageVersion: "0.1.0",
		}}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	e := got[0]
	if !e.Installed || e.Running {
		t.Fatalf("installed=%v running=%v", e.Installed, e.Running)
	}
	if e.Identification.Product == nil || e.Identification.Product.ID != identity.ProductOllamaID {
		t.Fatalf("product=%v", e.Identification.Product)
	}
	if e.Identification.Product.Category != identity.ProductCategoryLocalModelRuntime {
		t.Fatalf("category=%s", e.Identification.Product.Category)
	}
	if e.Identification.Confidence == identity.ConfidenceVerified {
		t.Fatal("unpinned catalog must not VERIFIED")
	}
}

func TestMonitorRuntimePayloadCategory(t *testing.T) {
	entry := InventoryEntry{
		Identity: identity.ApplicationIdentity{
			Path:       "/opt/homebrew/bin/ollama",
			Executable: "ollama",
		},
		Identification: identity.IdentificationResult{
			Product: &identity.KnownAIProduct{
				ID:       identity.ProductOllamaID,
				Name:     "Ollama",
				Vendor:   "Ollama",
				Category: identity.ProductCategoryLocalModelRuntime,
			},
			Confidence: identity.ConfidenceLow,
		},
		Installed: true,
	}
	art := artifactFromEntry(entry)
	if art.Payload["category"] != "local_model_runtime" {
		t.Fatalf("category=%v", art.Payload["category"])
	}
	if _, ok := art.Payload["ai_active"]; ok {
		t.Fatal("must not emit ai_active")
	}
}
