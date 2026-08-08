package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestMCPConfiguredInactive(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !mcpConfiguredForHost(home, identity.ProductCursorID) {
		t.Fatal("expected mcp configured")
	}
	// Presence ≠ active MCP sessions — we only set MCPConfigured flag.
}

func TestMCPConfiguredMissing(t *testing.T) {
	home := t.TempDir()
	if mcpConfiguredForHost(home, identity.ProductCursorID) {
		t.Fatal("expected false")
	}
}

func TestAttachExtensionCapabilitiesLocalModel(t *testing.T) {
	eng := NewEngine(EngineConfig{})
	byPath := map[string]*InventoryEntry{
		"ollama": {
			Serving: true,
			Identification: identity.IdentificationResult{
				Product: &identity.KnownAIProduct{ID: identity.ProductOllamaID, Category: identity.ProductCategoryLocalModelRuntime},
			},
		},
		"cline": {
			HostIDEProductID: identity.ProductCursorID,
			Identification: identity.IdentificationResult{
				Product: &identity.KnownAIProduct{ID: "cline", Category: identity.ProductCategoryAgenticIDEExtension},
			},
		},
	}
	eng.attachExtensionCapabilities(byPath)
	if byPath["cline"].LocalModelProductID != identity.ProductOllamaID {
		t.Fatalf("got %q", byPath["cline"].LocalModelProductID)
	}
}
