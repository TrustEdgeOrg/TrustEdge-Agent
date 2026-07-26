package security

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestSecurityMonitorBaselineThenNewArtifact(t *testing.T) {
	orig := listSecurityArtifacts
	defer func() { listSecurityArtifacts = orig }()

	base := []SecurityArtifact{{
		ID:          "service:existing",
		Type:        constants.TypeServiceInstall,
		Fingerprint: "existing",
		Payload:     map[string]any{"name": "existing"},
	}}
	next := append(base, SecurityArtifact{
		ID:          "driver:new",
		Type:        constants.TypeDriverLoad,
		Fingerprint: "new",
		Payload:     map[string]any{"name": "new"},
	})

	calls := 0
	listSecurityArtifacts = func() ([]SecurityArtifact, error) {
		calls++
		if calls == 1 {
			return base, nil
		}
		return next, nil
	}

	m := NewSecurityMonitor(nil)
	if got := m.Poll(); len(got) != 0 {
		t.Fatalf("baseline poll=%v want empty", got)
	}
	changes := m.Poll()
	if len(changes) != 1 {
		t.Fatalf("changes=%v want 1", changes)
	}
	if changes[0].Type != constants.TypeDriverLoad {
		t.Fatalf("type=%s", changes[0].Type)
	}
	if changes[0].Payload["name"] != "new" {
		t.Fatalf("payload=%v", changes[0].Payload)
	}
}

func TestSecurityMonitorChangedArtifact(t *testing.T) {
	orig := listSecurityArtifacts
	defer func() { listSecurityArtifacts = orig }()

	snapshots := [][]SecurityArtifact{
		{{
			ID:          `registry:hklm\run\updater`,
			Type:        constants.TypeRegistryPersist,
			Fingerprint: "old",
			Payload:     map[string]any{"value": "old.exe"},
		}},
		{{
			ID:          `registry:hklm\run\updater`,
			Type:        constants.TypeRegistryPersist,
			Fingerprint: "new",
			Payload:     map[string]any{"value": "new.exe"},
		}},
	}

	calls := 0
	listSecurityArtifacts = func() ([]SecurityArtifact, error) {
		out := snapshots[calls]
		calls++
		return out, nil
	}

	m := NewSecurityMonitor(nil)
	_ = m.Poll()
	changes := m.Poll()
	if len(changes) != 1 {
		t.Fatalf("changes=%v want 1", changes)
	}
	if changes[0].Type != constants.TypeRegistryPersist {
		t.Fatalf("type=%s", changes[0].Type)
	}
	if changes[0].Payload["value"] != "new.exe" {
		t.Fatalf("payload=%v", changes[0].Payload)
	}
}
