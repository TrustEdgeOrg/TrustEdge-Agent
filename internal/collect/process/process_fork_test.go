package process

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestProcessMonitorObservesFork(t *testing.T) {
	m := NewProcessMonitor(nil)
	// Silent baseline via Observe ready path: first Observe sets ready.
	ch := collect.Change{
		Type: constants.TypeProcessFork,
		Payload: map[string]any{
			"pid":        55,
			"ppid":       1,
			"comm":       "Cursor",
			"executable": "/Applications/Cursor.app/Contents/MacOS/Cursor",
		},
	}
	if !m.Observe(ch) {
		t.Fatal("first fork should be observed")
	}
	if m.Observe(ch) {
		t.Fatal("duplicate fork pid should be suppressed")
	}
}
