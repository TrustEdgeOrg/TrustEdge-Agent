package identity

import "testing"

func TestAIIDEExtensionCategories(t *testing.T) {
	if ProductCategoryAIIDEExtension != "ai_ide_extension" {
		t.Fatalf("got %q", ProductCategoryAIIDEExtension)
	}
	if ProductCategoryAgenticIDEExtension != "agentic_ide_extension" {
		t.Fatalf("got %q", ProductCategoryAgenticIDEExtension)
	}
}

func TestExtensionProductFields(t *testing.T) {
	p := KnownAIProduct{
		ID:                "example_ext",
		Name:              "Example Ext",
		Vendor:            "Example",
		Category:          ProductCategoryAIIDEExtension,
		ExtensionIDs:      []string{"example.extension"},
		HostIDEProductIDs: []string{ProductCursorID, "vscode"},
	}
	if len(p.ExtensionIDs) != 1 || p.ExtensionIDs[0] != "example.extension" {
		t.Fatalf("%+v", p.ExtensionIDs)
	}
	if p.Category == ProductCategoryCLIAgent || p.Category == ProductCategoryAgentRuntime {
		t.Fatal("must not collapse into agent taxonomy")
	}
}

func TestExtensionEvidenceKeys(t *testing.T) {
	keys := []EvidenceKey{
		EvidenceExtensionID,
		EvidenceExtensionPublisher,
		EvidenceExtensionPackage,
		EvidenceHostIDE,
		EvidenceExtensionEnabled,
		EvidenceExtensionActive,
		EvidenceMCPConfigured,
	}
	for _, k := range keys {
		if EvidenceLabel(k) == "" || EvidenceLabel(k) == string(k) && !containsUnderscore(string(k)) {
			// Label may fallback to underscored form; ensure non-empty.
		}
		if EvidenceLabel(k) == "" {
			t.Fatalf("empty label for %s", k)
		}
	}
}

func containsUnderscore(s string) bool {
	for _, r := range s {
		if r == '_' {
			return true
		}
	}
	return false
}

func TestDisplayNameAloneNotVerifiedExtension(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	// Without catalog extension entries yet, name-only must not VERIFIED.
	res := m.Identify(ApplicationIdentity{
		Path:              "/tmp/fake.cline/extension",
		Executable:        "Cline",
		PackageManager:    PackageManagerVSCodeExtension,
		PackageIdentifier: "fake.cline",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("spoof too strong: %s product=%v", res.Confidence, res.Product)
	}
}
