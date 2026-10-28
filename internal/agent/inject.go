package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

// InjectEvent is the drop-file schema for MCP gateway and Cursor governance hooks.
// Files land in <stateDir>/inject/*.json; the running agent uploads them
// with its enrolled device credentials (same path as live telemetry).
type InjectEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

func injectDir(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "inject")
}

func (a *Agent) runInjectWatcher(ctx context.Context, enqueue func(string, map[string]any)) {
	dir := injectDir(a.cfg.StatePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		a.log.Warn("inject dir create failed", "err", err, "dir", dir)
		return
	}
	a.log.Info("inject watcher active", "dir", dir)
	a.drainInjectDir(dir, enqueue)

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.drainInjectDir(dir, enqueue)
		}
	}
}

func (a *Agent) drainInjectDir(dir string, enqueue func(string, map[string]any)) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var ev InjectEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			if a.log != nil {
				a.log.Warn("inject parse failed", "path", path, "err", err)
			}
			_ = os.Remove(path)
			continue
		}
		typ := strings.TrimSpace(ev.Type)
		if typ == "" || ev.Payload == nil {
			_ = os.Remove(path)
			continue
		}
		switch typ {
		case constants.TypeFileOpen, constants.TypeProcessStart, constants.TypeProcessExit,
			constants.TypeNetworkConnection, constants.TypeClientDetails,
			constants.TypeAIAgentAction:
			enqueue(typ, ev.Payload)
			if a.log != nil {
				a.log.Info("inject event enqueued", "type", typ, "file", ent.Name())
			}
		default:
			if a.log != nil {
				a.log.Warn("inject unsupported type", "type", typ, "file", ent.Name())
			}
		}
		_ = os.Remove(path)
	}
}
