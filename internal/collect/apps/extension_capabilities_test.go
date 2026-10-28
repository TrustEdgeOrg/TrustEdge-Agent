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

func TestMCPServersForHost(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{
  "mcpServers": {
    "github": {"command": "docker"},
    "slack": {"command": "npx"}
  },
  "env": {"SECRET": "must-not-appear"}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := mcpServersForHost(home, identity.ProductCursorID)
	if len(got) != 2 || got[0] != "github" || got[1] != "slack" {
		t.Fatalf("servers=%v", got)
	}
}

func TestAttachExtensionCapabilitiesMCPServers(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"mcpServers":{"github":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	origHome := homeDirFn
	t.Cleanup(func() { homeDirFn = origHome })
	homeDirFn = func() (string, error) { return home, nil }

	eng := NewEngine(EngineConfig{})
	byPath := map[string]*InventoryEntry{
		"cline": {
			HostIDEProductID: identity.ProductCursorID,
			Identification: identity.IdentificationResult{
				Product: &identity.KnownAIProduct{ID: "cline", Category: identity.ProductCategoryAgenticIDEExtension},
			},
		},
	}
	eng.attachExtensionCapabilities(byPath)
	if !byPath["cline"].MCPConfigured {
		t.Fatal("expected mcp configured")
	}
	if len(byPath["cline"].MCPServers) != 1 || byPath["cline"].MCPServers[0] != "github" {
		t.Fatalf("servers=%v", byPath["cline"].MCPServers)
	}
}

func TestAttachExtensionCapabilitiesMCPServersOnHostIDE(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"mcpServers":{"github":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	origHome := homeDirFn
	t.Cleanup(func() { homeDirFn = origHome })
	homeDirFn = func() (string, error) { return home, nil }

	eng := NewEngine(EngineConfig{})
	byPath := map[string]*InventoryEntry{
		"cursor": {
			Identification: identity.IdentificationResult{
				Product: &identity.KnownAIProduct{ID: identity.ProductCursorID, Category: identity.ProductCategoryCodeEditor},
			},
		},
	}
	eng.attachExtensionCapabilities(byPath)
	if !byPath["cursor"].MCPConfigured {
		t.Fatal("expected mcp configured on cursor host")
	}
	if len(byPath["cursor"].MCPServers) != 1 || byPath["cursor"].MCPServers[0] != "github" {
		t.Fatalf("servers=%v", byPath["cursor"].MCPServers)
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
