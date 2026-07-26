package security

import (
	"log"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
)

// SecurityArtifact is a point-in-time host security artifact found by polling.
type SecurityArtifact struct {
	ID          string
	Type        string
	Fingerprint string
	Payload     map[string]any
}

// SecurityMonitor reports new or changed security lifecycle artifacts.
type SecurityMonitor struct {
	Logger *log.Logger
	mu     sync.Mutex
	seen   map[string]SecurityArtifact
	ready  bool
}

func NewSecurityMonitor(logger *log.Logger) *SecurityMonitor {
	return &SecurityMonitor{
		Logger: logger,
		seen:   map[string]SecurityArtifact{},
	}
}

func (m *SecurityMonitor) Poll() []collect.Change {
	artifacts, err := listSecurityArtifacts()
	if err != nil {
		m.logf("security poll: %v", err)
		return nil
	}
	current := make(map[string]SecurityArtifact, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.ID == "" || artifact.Type == "" {
			continue
		}
		current[artifact.ID] = artifact
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.ready {
		m.seen = current
		m.ready = true
		return nil
	}

	var changes []collect.Change
	for id, artifact := range current {
		prev, ok := m.seen[id]
		if ok && prev.Fingerprint == artifact.Fingerprint {
			continue
		}
		changes = append(changes, collect.Change{
			Type:    artifact.Type,
			Payload: artifact.Payload,
		})
	}
	m.seen = current
	return changes
}

func (m *SecurityMonitor) logf(format string, args ...any) {
	if m.Logger != nil {
		m.Logger.Printf(format, args...)
	}
}
