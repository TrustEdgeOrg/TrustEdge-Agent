package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestParseDisabledExtensionIDs(t *testing.T) {
	raw := []byte(`{
  "extensionsIdentifiers/disabled": [
    {"id": "saoudrizwan.claude-dev"},
    {"id": "Continue.continue"}
  ]
}`)
	ids, ok := parseDisabledExtensionIDs(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	if _, off := ids["saoudrizwan.claude-dev"]; !off {
		t.Fatal("missing cline")
	}
	if _, off := ids["continue.continue"]; !off {
		t.Fatal("missing continue")
	}
}

func TestParseDisabledExtensionIDsEmptyKnown(t *testing.T) {
	ids, ok := parseDisabledExtensionIDs([]byte(`{"foo":1}`))
	if !ok || len(ids) != 0 {
		t.Fatalf("ok=%v len=%d", ok, len(ids))
	}
}

func TestApplyExtensionEnabledState(t *testing.T) {
	entry := &InventoryEntry{
		ExtensionID: "saoudrizwan.claude-dev",
		Identification: identity.IdentificationResult{
			Product: &identity.KnownAIProduct{ID: "cline"},
		},
	}
	disabled := map[string]struct{}{"saoudrizwan.claude-dev": {}}
	applyExtensionEnabledState(entry, disabled, true)
	if entry.Enabled == nil || *entry.Enabled {
		t.Fatal("expected disabled")
	}
	applyExtensionEnabledState(entry, map[string]struct{}{}, true)
	if entry.Enabled == nil || !*entry.Enabled {
		t.Fatal("expected enabled")
	}
	applyExtensionEnabledState(entry, nil, false)
	if entry.Enabled != nil {
		t.Fatal("unknown when state unread")
	}
}
