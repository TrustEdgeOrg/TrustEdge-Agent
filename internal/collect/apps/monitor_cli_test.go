package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestArtifactIncludesCLIFields(t *testing.T) {
	entry := InventoryEntry{
		Identity: identity.ApplicationIdentity{
			Path:              "/opt/homebrew/Cellar/claude-code/1.0/bin/claude",
			Executable:        "claude",
			InvocationPath:    "/opt/homebrew/bin/claude",
			ResolvedPath:      "/opt/homebrew/Cellar/claude-code/1.0/bin/claude",
			PackageManager:    "homebrew",
			PackageIdentifier: "claude-code",
			PackageVersion:    "1.0",
			EntryPoint:        "claude",
		},
		Identification: identity.IdentificationResult{
			Product: &identity.KnownAIProduct{
				ID:       identity.ProductClaudeCodeID,
				Name:     "Claude Code",
				Vendor:   "Anthropic",
				Category: identity.ProductCategoryCLIAgent,
			},
			Confidence: identity.ConfidenceLow,
		},
		Installed: true,
	}
	art := artifactFromEntry(entry)
	if art.Payload["invocation_path"] != "/opt/homebrew/bin/claude" {
		t.Fatalf("invocation_path=%v", art.Payload["invocation_path"])
	}
	if art.Payload["resolved_path"] != "/opt/homebrew/Cellar/claude-code/1.0/bin/claude" {
		t.Fatalf("resolved_path=%v", art.Payload["resolved_path"])
	}
	if art.Payload["package_manager"] != "homebrew" {
		t.Fatalf("package_manager=%v", art.Payload["package_manager"])
	}
	if art.Payload["package_identifier"] != "claude-code" {
		t.Fatalf("package_identifier=%v", art.Payload["package_identifier"])
	}
	if art.Payload["entry_point"] != "claude" {
		t.Fatalf("entry_point=%v", art.Payload["entry_point"])
	}
	if art.Payload["version"] != "1.0" {
		t.Fatalf("version=%v", art.Payload["version"])
	}
	if art.Payload["category"] != "cli_agent" {
		t.Fatalf("category=%v", art.Payload["category"])
	}
}
