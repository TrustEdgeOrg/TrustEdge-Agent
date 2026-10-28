package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
)

// WriteInject drops an ai_agent_action into the agent's inject directory
// so the existing EventBatcher upload path is reused (no second pipeline).
func WriteInject(injectDir string, action agentaction.AgentAction) error {
	if err := os.MkdirAll(injectDir, 0o700); err != nil {
		return err
	}
	env := map[string]any{
		"type":    agentaction.EventType,
		"payload": action.Payload(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("mcp-%d-%s.json", time.Now().UnixNano(), action.CorrelationID)
	path := filepath.Join(injectDir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DefaultInjectDir returns <stateDir>/inject given a state.json path.
func DefaultInjectDir(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "inject")
}
