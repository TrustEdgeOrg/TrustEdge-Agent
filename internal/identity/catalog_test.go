package identity

import "testing"

func TestDefaultCatalogContainsVerifiedCursor(t *testing.T) {
	c := DefaultCatalog()
	p, ok := c.Lookup(ProductCursorID)
	if !ok {
		t.Fatal("cursor missing from catalog")
	}
	if p.Name != "Cursor" {
		t.Fatalf("Name=%q", p.Name)
	}
	if p.Category != ProductCategoryCodeEditor {
		t.Fatalf("Category=%q", p.Category)
	}
	if !contains(p.BundleIDs, "com.todesktop.230313mzl4w4u92") {
		t.Fatalf("BundleIDs=%v missing verified id", p.BundleIDs)
	}
	if !contains(p.SigningIdentifiers, "com.todesktop.230313mzl4w4u92") {
		t.Fatalf("SigningIdentifiers=%v", p.SigningIdentifiers)
	}
	if !contains(p.TeamIDs, "VDXQ22DGB9") {
		t.Fatalf("TeamIDs=%v", p.TeamIDs)
	}
	if !contains(p.CandidateNames, "Cursor") {
		t.Fatalf("CandidateNames=%v", p.CandidateNames)
	}
	if len(p.ExpectedHashes) != 0 {
		t.Fatal("ExpectedHashes must remain unresolved/empty until pinned")
	}
}

func TestDefaultCatalogContainsVerifiedClaude(t *testing.T) {
	c := DefaultCatalog()
	p, ok := c.Lookup(ProductClaudeID)
	if !ok {
		t.Fatal("claude missing from catalog")
	}
	if p.Name != "Claude" || p.Vendor != "Anthropic" {
		t.Fatalf("Name=%q Vendor=%q", p.Name, p.Vendor)
	}
	if p.Category != ProductCategoryChatClient {
		t.Fatalf("Category=%q", p.Category)
	}
	if !contains(p.BundleIDs, "com.anthropic.claudefordesktop") {
		t.Fatalf("BundleIDs=%v", p.BundleIDs)
	}
	if !contains(p.SigningIdentifiers, "com.anthropic.claudefordesktop") {
		t.Fatalf("SigningIdentifiers=%v", p.SigningIdentifiers)
	}
	if !contains(p.TeamIDs, "Q6L2SF6YDW") {
		t.Fatalf("TeamIDs=%v", p.TeamIDs)
	}
	if !contains(p.CandidateNames, "Claude") {
		t.Fatalf("CandidateNames=%v", p.CandidateNames)
	}
	if len(p.ExpectedHashes) != 0 {
		t.Fatal("ExpectedHashes must remain unresolved/empty until pinned")
	}
}

func TestCatalogLookupMissing(t *testing.T) {
	if _, ok := DefaultCatalog().Lookup("does-not-exist"); ok {
		t.Fatal("expected miss")
	}
}

func TestNewCatalogCopy(t *testing.T) {
	c := NewCatalog(KnownAIProduct{ID: "a", Name: "A"})
	prods := c.Products()
	prods[0].Name = "mutated"
	if c.Products()[0].Name != "A" {
		t.Fatal("Products must return a copy")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
