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

func TestDefaultCatalogContainsCLIAgents(t *testing.T) {
	c := DefaultCatalog()
	cases := []struct {
		id, name, vendor, cmd string
	}{
		{ProductClaudeCodeID, "Claude Code", "Anthropic", "claude"},
		{ProductCodexCLIID, "OpenAI Codex CLI", "OpenAI", "codex"},
		{ProductGeminiCLIID, "Gemini CLI", "Google", "gemini"},
		{ProductCopilotCLIID, "GitHub Copilot CLI", "GitHub", "copilot"},
		{ProductOpenCodeID, "OpenCode", "OpenCode", "opencode"},
	}
	for _, tc := range cases {
		p, ok := c.Lookup(tc.id)
		if !ok {
			t.Fatalf("%s missing from catalog", tc.id)
		}
		if p.Category != ProductCategoryCLIAgent {
			t.Fatalf("%s Category=%q", tc.id, p.Category)
		}
		if p.Name != tc.name || p.Vendor != tc.vendor {
			t.Fatalf("%s Name=%q Vendor=%q", tc.id, p.Name, p.Vendor)
		}
		if !contains(p.CandidateNames, tc.cmd) && !contains(p.ExecutableNames, tc.cmd) {
			t.Fatalf("%s missing command %q", tc.id, tc.cmd)
		}
		if len(p.PackageIdentifiers) != 0 || len(p.PackageManagers) != 0 {
			t.Fatalf("%s package fields must remain unresolved", tc.id)
		}
		if len(p.BundleIDs) != 0 || len(p.TeamIDs) != 0 || len(p.SigningIdentifiers) != 0 {
			t.Fatalf("%s must not invent signing/bundle evidence", tc.id)
		}
	}
}

func TestCLIAgentNameOnlyIsLowNotVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:       "/tmp/claude",
		Executable: "claude",
	})
	if res.Product == nil || res.Product.ID != ProductClaudeCodeID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence != ConfidenceLow {
		t.Fatalf("name-only CLI must be LOW, got %s", res.Confidence)
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
