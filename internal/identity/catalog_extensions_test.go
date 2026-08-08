package identity

import "testing"

func TestCatalogContainsVSCodeAndExtensions(t *testing.T) {
	c := DefaultCatalog()
	if _, ok := c.Lookup(ProductVSCodeID); !ok {
		t.Fatal("missing vscode")
	}
	for _, id := range []string{"github_copilot", "continue", "cline", "roo_code"} {
		p, ok := c.Lookup(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if len(p.ExtensionIDs) == 0 {
			t.Fatalf("%s missing ExtensionIDs", id)
		}
	}
	cline, _ := c.Lookup("cline")
	if cline.Category != ProductCategoryAgenticIDEExtension {
		t.Fatalf("cline category=%s", cline.Category)
	}
	copilot, _ := c.Lookup("github_copilot")
	if copilot.Category != ProductCategoryAIIDEExtension {
		t.Fatalf("copilot category=%s", copilot.Category)
	}
}

func TestMatcherVerifiedExtensionID(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/tmp/ext/saoudrizwan.claude-dev-3.0.0",
		PackageManager:    PackageManagerVSCodeExtension,
		PackageIdentifier: "saoudrizwan.claude-dev",
		Version:           "3.0.0",
		Executable:        "claude-dev",
	})
	if res.Product == nil || res.Product.ID != "cline" {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence != ConfidenceVerified && res.Confidence != ConfidenceHigh {
		t.Fatalf("confidence=%s matched=%v", res.Confidence, res.Matched)
	}
}

func TestMatcherWrongPublisherNotVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/tmp/ext/fake.claude-dev-1.0.0",
		PackageManager:    PackageManagerVSCodeExtension,
		PackageIdentifier: "fake.claude-dev",
		Executable:        "claude-dev",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("wrong publisher too strong: %s product=%v", res.Confidence, res.Product)
	}
}

func TestMatcherDisplayNameAloneLow(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:           "/tmp/ext/evil.cline-1.0.0",
		PackageManager: PackageManagerCursorExtension,
		Executable:     "Cline",
		// No PackageIdentifier / wrong id
		PackageIdentifier: "evil.cline",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("displayName spoof too strong: %s", res.Confidence)
	}
}

func TestMatcherNonAIExtensionIgnored(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/tmp/ext/ms-python.python-2024.0.0",
		PackageManager:    PackageManagerVSCodeExtension,
		PackageIdentifier: "ms-python.python",
		Executable:        "python",
	})
	if res.Product != nil && (res.Product.Category == ProductCategoryAIIDEExtension || res.Product.Category == ProductCategoryAgenticIDEExtension) {
		t.Fatalf("python must not match AI extension: %v", res.Product)
	}
}
