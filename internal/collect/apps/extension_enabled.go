package apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// readDisabledExtensionIDs returns lowercase extension ids disabled for a host IDE.
// Returns ok=false when the state file is missing or unreadable (ENABLED=UNKNOWN).
func readDisabledExtensionIDs(home string, hostProductID string) (map[string]struct{}, bool) {
	path := disabledExtensionsStatePath(home, hostProductID)
	if path == "" {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	ids, ok := parseDisabledExtensionIDs(raw)
	return ids, ok
}

func disabledExtensionsStatePath(home, hostProductID string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	switch strings.ToLower(hostProductID) {
	case identity.ProductCursorID:
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "storage.json")
	case identity.ProductVSCodeID:
		return filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "storage.json")
	default:
		return ""
	}
}

func parseDisabledExtensionIDs(raw []byte) (map[string]struct{}, bool) {
	// storage.json is a flat string-keyed JSON object; disabled extensions typically
	// live under "extensionsIdentifiers/disabled" as a JSON-encoded array string
	// or a native array of {id: "publisher.name"} objects.
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, false
	}
	blob, ok := root["extensionsIdentifiers/disabled"]
	if !ok || len(blob) == 0 {
		// No disabled list → treat as readable empty set (all installed are enabled).
		return map[string]struct{}{}, true
	}
	ids := decodeDisabledIDList(blob)
	if ids == nil {
		return nil, false
	}
	return ids, true
}

func decodeDisabledIDList(blob json.RawMessage) map[string]struct{} {
	out := make(map[string]struct{})
	// Native array form.
	var objs []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(blob, &objs); err == nil {
		for _, o := range objs {
			id := strings.ToLower(strings.TrimSpace(o.ID))
			if id != "" {
				out[id] = struct{}{}
			}
		}
		return out
	}
	// Sometimes stored as a JSON string containing the array.
	var asString string
	if err := json.Unmarshal(blob, &asString); err == nil && asString != "" {
		return decodeDisabledIDList(json.RawMessage(asString))
	}
	return nil
}

func applyExtensionEnabledState(entry *InventoryEntry, disabled map[string]struct{}, stateKnown bool) {
	if entry == nil || !stateKnown {
		entry.Enabled = nil
		return
	}
	id := strings.ToLower(strings.TrimSpace(entry.ExtensionID))
	if id == "" {
		id = strings.ToLower(strings.TrimSpace(entry.Identity.PackageIdentifier))
	}
	enabled := true
	if _, off := disabled[id]; off {
		enabled = false
	}
	entry.Enabled = &enabled
	if enabled {
		if !hasEvidence(entry.Identification.Matched, identity.EvidenceExtensionEnabled) {
			entry.Identification.Matched = append(entry.Identification.Matched, identity.EvidenceExtensionEnabled)
		}
	} else if !hasEvidence(entry.Identification.Failed, identity.EvidenceExtensionEnabled) {
		entry.Identification.Failed = append(entry.Identification.Failed, identity.EvidenceExtensionEnabled)
	}
}
