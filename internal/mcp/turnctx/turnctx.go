package turnctx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const activeTurnFile = ".active-turn.json"

// ActiveTurn links a Cursor conversation turn to subsequent MCP gateway emissions.
type ActiveTurn struct {
	SessionID string    `json:"session_id"`
	TurnID    string    `json:"turn_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Write stores the active turn for MCP gateway correlation.
func Write(injectDir, sessionID, turnID string, ts time.Time) error {
	sessionID = strings.TrimSpace(sessionID)
	turnID = strings.TrimSpace(turnID)
	if sessionID == "" || turnID == "" {
		return nil
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	if err := os.MkdirAll(injectDir, 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(ActiveTurn{
		SessionID: sessionID,
		TurnID:    turnID,
		Timestamp: ts.UTC(),
	})
	if err != nil {
		return err
	}
	path := filepath.Join(injectDir, activeTurnFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Read returns the active turn when present and not older than maxAge.
func Read(injectDir string, maxAge time.Duration) (ActiveTurn, bool) {
	if maxAge <= 0 {
		maxAge = 30 * time.Minute
	}
	path := filepath.Join(injectDir, activeTurnFile)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ActiveTurn{}, false
	}
	var turn ActiveTurn
	if err := json.Unmarshal(data, &turn); err != nil {
		return ActiveTurn{}, false
	}
	turn.SessionID = strings.TrimSpace(turn.SessionID)
	turn.TurnID = strings.TrimSpace(turn.TurnID)
	if turn.SessionID == "" || turn.TurnID == "" {
		return ActiveTurn{}, false
	}
	if turn.Timestamp.IsZero() {
		return turn, true
	}
	if time.Since(turn.Timestamp.UTC()) > maxAge {
		return ActiveTurn{}, false
	}
	return turn, true
}
