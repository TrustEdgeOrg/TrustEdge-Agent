package apps

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// Monitor emits known_ai_app inventory deltas (security-style silent baseline).
type Monitor struct {
	Logger *log.Logger
	Engine *Engine

	mu    sync.Mutex
	seen  map[string]string // id -> fingerprint
	ready bool
}

// NewMonitor constructs a known-AI inventory monitor.
func NewMonitor(logger *log.Logger, engine *Engine) *Monitor {
	if engine == nil {
		engine = NewEngine(EngineConfig{Logger: logger})
	}
	return &Monitor{
		Logger: logger,
		Engine: engine,
		seen:   map[string]string{},
	}
}

// Poll discovers installed/running known AI apps and returns changed events.
func (m *Monitor) Poll() []collect.Change {
	entries, err := m.Engine.Inventory()
	if err != nil {
		m.logf("known-ai poll: %v", err)
		return nil
	}

	current := make(map[string]inventoryArtifact, len(entries))
	for _, entry := range entries {
		art := artifactFromEntry(entry)
		if art.ID == "" {
			continue
		}
		current[art.ID] = art
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.ready {
		m.seen = fingerprints(current)
		m.ready = true
		return nil
	}

	var changes []collect.Change
	for id, art := range current {
		prev, ok := m.seen[id]
		if ok && prev == art.Fingerprint {
			continue
		}
		changes = append(changes, collect.Change{
			Type:    constants.TypeKnownAIApp,
			Payload: art.Payload,
		})
	}
	// Removals: previously seen product install no longer present.
	for id := range m.seen {
		if _, ok := current[id]; ok {
			continue
		}
		changes = append(changes, collect.Change{
			Type: constants.TypeKnownAIApp,
			Payload: map[string]any{
				"id":        id,
				"removed":   true,
				"installed": false,
				"running":   false,
			},
		})
	}
	m.seen = fingerprints(current)
	return changes
}

type inventoryArtifact struct {
	ID          string
	Fingerprint string
	Payload     map[string]any
}

func artifactFromEntry(entry InventoryEntry) inventoryArtifact {
	p := entry.Identification.Product
	if p == nil {
		return inventoryArtifact{}
	}
	path := posixPath(entry.Identity.Path)
	id := fmt.Sprintf("%s:%s", p.ID, pathKey(path))
	matched := evidenceStrings(entry.Identification.Matched)
	failed := evidenceStrings(entry.Identification.Failed)
	payload := map[string]any{
		"id":               id,
		"product_id":       p.ID,
		"product_name":     p.Name,
		"vendor":           p.Vendor,
		"category":         string(p.Category),
		"confidence":       string(entry.Identification.Confidence),
		"installed":        entry.Installed,
		"running":          entry.Running,
		"path":             path,
		"bundle_id":        entry.Identity.BundleID,
		"version":          entry.Identity.Version,
		"executable":       entry.Identity.Executable,
		"signing_id":       entry.Identity.SigningIdentifier,
		"team_id":          entry.Identity.TeamID,
		"signature_valid":  entry.Identity.SignatureValid,
		"matched_evidence": matched,
		"failed_evidence":  failed,
		"pids":             intsToAny(entry.PIDs),
	}
	fp := fingerprintPayload(payload)
	return inventoryArtifact{ID: id, Fingerprint: fp, Payload: payload}
}

func fingerprints(m map[string]inventoryArtifact) map[string]string {
	out := make(map[string]string, len(m))
	for id, art := range m {
		out[id] = art.Fingerprint
	}
	return out
}

func evidenceStrings(keys []identity.EvidenceKey) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	return out
}

func intsToAny(v []int) []any {
	out := make([]any, len(v))
	for i, n := range v {
		out[i] = n
	}
	return out
}

func fingerprintPayload(p map[string]any) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		if k == "pids" {
			continue // PID churn shouldn't spam events; running bool covers presence
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v;", k, p[k])
	}
	// Include sorted PID list length + set so new helpers still update running detail carefully:
	// actually plan wants running true/false; PID changes alone are noisy. Omit pids from fp.
	return b.String()
}

func (m *Monitor) logf(format string, args ...any) {
	if m.Logger != nil {
		m.Logger.Printf(format, args...)
	}
}
