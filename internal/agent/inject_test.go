package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestDrainInjectDir(t *testing.T) {
	dir := t.TempDir()
	inject := filepath.Join(dir, "inject")
	if err := os.MkdirAll(inject, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := InjectEvent{
		Type: constants.TypeFileOpen,
		Payload: map[string]any{
			"pid":  1,
			"comm": "Cursor",
			"path": "/tmp/.env",
		},
	}
	raw, _ := json.Marshal(payload)
	path := filepath.Join(inject, "one.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	a := &Agent{}
	var gotType string
	var gotPayload map[string]any
	a.drainInjectDir(inject, func(typ string, p map[string]any) {
		gotType = typ
		gotPayload = p
	})
	if gotType != constants.TypeFileOpen {
		t.Fatalf("type=%q", gotType)
	}
	if gotPayload["path"] != "/tmp/.env" {
		t.Fatalf("payload=%v", gotPayload)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("inject file should be removed after enqueue")
	}
}

func TestDrainInjectAIAgentAction(t *testing.T) {
	dir := t.TempDir()
	inject := filepath.Join(dir, "inject")
	if err := os.MkdirAll(inject, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := InjectEvent{
		Type: constants.TypeAIAgentAction,
		Payload: map[string]any{
			"agent_type": "claude_code",
			"channel":    "MCP",
			"tool":       "query_customers",
			"governed":   true,
		},
	}
	raw, _ := json.Marshal(payload)
	path := filepath.Join(inject, "mcp.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{}
	var gotType string
	a.drainInjectDir(inject, func(typ string, _ map[string]any) {
		gotType = typ
	})
	if gotType != constants.TypeAIAgentAction {
		t.Fatalf("type=%q", gotType)
	}
}
