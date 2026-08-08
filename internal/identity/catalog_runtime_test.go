package identity

import "testing"

func TestLocalModelRuntimeCategory(t *testing.T) {
	if ProductCategoryLocalModelRuntime != "local_model_runtime" {
		t.Fatalf("category=%q", ProductCategoryLocalModelRuntime)
	}
}

func TestDefaultCatalogContainsRuntimes(t *testing.T) {
	c := DefaultCatalog()
	for _, tc := range []struct {
		id   string
		name string
		exe  string
	}{
		{ProductOllamaID, "Ollama", "ollama"},
		{ProductLlamaCppID, "llama.cpp", "llama-server"},
	} {
		p, ok := c.Lookup(tc.id)
		if !ok {
			t.Fatalf("missing %s", tc.id)
		}
		if p.Name != tc.name || p.Category != ProductCategoryLocalModelRuntime {
			t.Fatalf("%+v", p)
		}
		if p.Category == ProductCategoryCLIAgent {
			t.Fatal("runtime must not be cli_agent")
		}
		if len(p.PackageIdentifiers) != 0 || len(p.DefaultLocalEndpoints) != 0 {
			t.Fatalf("%s must not invent unresolved package/endpoint fields", tc.id)
		}
		found := false
		for _, n := range p.ExecutableNames {
			if n == tc.exe {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing executable %s", tc.id, tc.exe)
		}
	}
	p, _ := c.Lookup(ProductLlamaCppID)
	if p.RuntimeFamily != RuntimeFamilyLlamaCppCompatible {
		t.Fatalf("RuntimeFamily=%q", p.RuntimeFamily)
	}
}

func TestOllamaNameOnlyIsLowNotVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:       "/tmp/ollama",
		Executable: "ollama",
	})
	if res.Product == nil || res.Product.ID != ProductOllamaID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Product.Category != ProductCategoryLocalModelRuntime {
		t.Fatalf("category=%s", res.Product.Category)
	}
	if res.Confidence != ConfidenceLow {
		t.Fatalf("name-only runtime must be LOW, got %s", res.Confidence)
	}
	if res.Confidence == ConfidenceVerified {
		t.Fatal("must never VERIFIED from name alone")
	}
}

func TestRuntimeEvidenceKeysPresent(t *testing.T) {
	keys := []EvidenceKey{
		EvidenceListener,
		EvidenceListenerExposure,
		EvidenceRuntimeFingerprint,
		EvidenceModelArtifact,
		EvidenceLocalClient,
	}
	for _, k := range keys {
		if EvidenceLabel(k) == "" {
			t.Fatalf("empty label for %s", k)
		}
	}
}
