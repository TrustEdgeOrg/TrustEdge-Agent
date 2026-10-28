package events_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/events"
)

func TestWriteMCPHostBridge_github(t *testing.T) {
	dir := t.TempDir()
	if err := events.WriteMCPHostBridge(dir, "corr-1", "github", "list_issues"); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "mcp-host-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 host bridge files, got %d", len(files))
	}

	var types []string
	for _, f := range files {
		var env struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		types = append(types, env.Type)
		if env.Payload["correlation_id"] != "corr-1" {
			t.Fatalf("correlation_id mismatch in %s", f)
		}
		if env.Payload["source"] != events.MCPGatewayHostBridgeSource {
			t.Fatalf("source mismatch in %s", f)
		}
	}
	if !contains(types, constants.TypeNetworkConnection) || !contains(types, constants.TypeProcessStart) {
		t.Fatalf("unexpected event types: %v", types)
	}

	var netPayload map[string]any
	for _, f := range files {
		if !strings.Contains(f, "-net-") {
			continue
		}
		raw, _ := os.ReadFile(f)
		var env struct {
			Payload map[string]any `json:"payload"`
		}
		_ = json.Unmarshal(raw, &env)
		netPayload = env.Payload
	}
	if netPayload["domain"] != "api.github.com" {
		t.Fatalf("domain = %v", netPayload["domain"])
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
