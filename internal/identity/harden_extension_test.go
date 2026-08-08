package identity

import "testing"

func TestHardenExtensionNeverAgentActiveTaxonomy(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/x/saoudrizwan.claude-dev-1",
		PackageManager:    PackageManagerVSCodeExtension,
		PackageIdentifier: "saoudrizwan.claude-dev",
	})
	if res.Product == nil {
		t.Fatal("expected cline")
	}
	if res.Product.Category == ProductCategoryCLIAgent || res.Product.Category == ProductCategoryAgentRuntime {
		t.Fatal("must not classify as CLI/agent_runtime")
	}
	if res.Product.Category != ProductCategoryAgenticIDEExtension {
		t.Fatalf("got %s", res.Product.Category)
	}
}
