package identity

import "testing"

func TestHardenCLISpoofFilenameNeverVerified(t *testing.T) {
	res := NewMatcher(DefaultCatalog()).Identify(ApplicationIdentity{
		Executable: "claude",
		Path:       "/tmp/claude",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("spoof too strong: %s", res.Confidence)
	}
}

func TestHardenCLIWrongPackageFailsStrongMatch(t *testing.T) {
	cat := NewCatalog(KnownAIProduct{
		ID:                 "x",
		Name:               "X",
		Category:           ProductCategoryCLIAgent,
		ExecutableNames:    []string{"claude"},
		PackageManagers:    []string{"npm"},
		PackageIdentifiers: []string{"@good/pkg"},
		EntryPoints:        []string{"cli.js"},
	})
	res := NewMatcher(cat).Identify(ApplicationIdentity{
		Executable:        "claude",
		PackageManager:    "npm",
		PackageIdentifier: "@evil/pkg",
		EntryPoint:        "cli.js",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("wrong package: %s", res.Confidence)
	}
}
