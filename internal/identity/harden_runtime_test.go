package identity

import "testing"

func TestHardenRuntimeNeverCLIAgent(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{Executable: "ollama", Path: "/opt/homebrew/bin/ollama"})
	if res.Product == nil {
		t.Fatal("expected product")
	}
	if res.Product.Category != ProductCategoryLocalModelRuntime {
		t.Fatalf("got %s", res.Product.Category)
	}
	if res.Product.Category == ProductCategoryCLIAgent || res.Product.Category == ProductCategoryAgentRuntime {
		t.Fatal("runtime must not be agent taxonomy")
	}
}

func TestHardenRenamedLlamaCppConservative(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Executable: "my-llm-server",
		Path:       "/usr/local/bin/my-llm-server",
	})
	if res.Product != nil {
		t.Fatalf("renamed binary without catalog name must not match: %v", res.Product)
	}
}
