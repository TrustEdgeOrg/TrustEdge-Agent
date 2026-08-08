package identity

import "testing"

func TestCLINameOnlyNeverVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Executable:     "claude",
		ExecutablePath: "/tmp/claude",
		Path:           "/tmp/claude",
		InvocationPath: "/tmp/claude",
		ResolvedPath:   "/tmp/claude",
	})
	if res.Product == nil || res.Product.ID != ProductClaudeCodeID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence != ConfidenceLow {
		t.Fatalf("want LOW, got %s", res.Confidence)
	}
	if res.Confidence == ConfidenceVerified {
		t.Fatal("filename alone must never VERIFIED")
	}
}

func TestCLIWrongPackageIdentityFails(t *testing.T) {
	cat := NewCatalog(KnownAIProduct{
		ID:                 "fake_cli",
		Name:               "Fake CLI",
		Category:           ProductCategoryCLIAgent,
		ExecutableNames:    []string{"claude"},
		PackageManagers:    []string{"npm"},
		PackageIdentifiers: []string{"@good/claude"},
		EntryPoints:        []string{"claude.js"},
	})
	m := NewMatcher(cat)
	res := m.Identify(ApplicationIdentity{
		Executable:        "claude",
		Path:              "/opt/homebrew/bin/claude",
		ResolvedPath:      "/opt/npm/pkg/bin/claude.js",
		PackageManager:    "npm",
		PackageIdentifier: "@evil/claude",
		EntryPoint:        "claude.js",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("wrong package must not be strong: %s", res.Confidence)
	}
	if !hasEvidence(res.Failed, EvidencePackageIdentity) {
		t.Fatal("expected package_identity failed")
	}
}

func TestCLIVerifiedWithPackageAndProvenance(t *testing.T) {
	cat := NewCatalog(KnownAIProduct{
		ID:                 "pinned_cli",
		Name:               "Pinned CLI",
		Category:           ProductCategoryCLIAgent,
		ExecutableNames:    []string{"claude"},
		PackageManagers:    []string{"homebrew"},
		PackageIdentifiers: []string{"claude-code"},
		EntryPoints:        []string{"claude"},
	})
	m := NewMatcher(cat)
	res := m.Identify(ApplicationIdentity{
		Executable:        "claude",
		Path:              "/opt/homebrew/Cellar/claude-code/1.0.0/bin/claude",
		ResolvedPath:      "/opt/homebrew/Cellar/claude-code/1.0.0/bin/claude",
		PackageManager:    "homebrew",
		PackageIdentifier: "claude-code",
		EntryPoint:        "claude",
	})
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("want VERIFIED, got %s matched=%v failed=%v", res.Confidence, res.Matched, res.Failed)
	}
}

func TestNodeInterpreterAloneNotProduct(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Executable:     "node",
		ExecutablePath: "/usr/local/bin/node",
		Path:           "/usr/local/bin/node",
		Interpreter:    "node",
	})
	if res.Product != nil || res.Confidence != ConfidenceUnknown {
		t.Fatalf("node alone must not match: product=%v conf=%s", res.Product, res.Confidence)
	}
}

func TestStubCatalogUnreachableVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Executable:        "codex",
		Path:              "/opt/homebrew/Cellar/codex/0.1/bin/codex",
		ResolvedPath:      "/opt/homebrew/Cellar/codex/0.1/bin/codex",
		PackageManager:    "homebrew",
		PackageIdentifier: "codex",
		EntryPoint:        "codex",
	})
	if res.Product == nil || res.Product.ID != ProductCodexCLIID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence == ConfidenceVerified {
		t.Fatal("stub catalog without PackageIdentifiers must not reach VERIFIED")
	}
}
