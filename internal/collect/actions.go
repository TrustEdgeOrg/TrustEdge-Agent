package collect

import (
	"sync"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// ActionTracker accumulates foreground app focus between action_summary posts.
// Sample may run on a fast ticker while SnapshotAndReset runs on the slower
// action interval; methods are safe for concurrent use.
type ActionTracker struct {
	clock     clock.Clock
	probe     PlatformProbe
	pollEvery time.Duration

	mu          sync.Mutex
	lastApp     string
	switches    int
	focus       map[string]*models.AppFocus
	windowStart time.Time
}

func NewActionTracker(clk clock.Clock, probe PlatformProbe, pollEvery time.Duration) *ActionTracker {
	if clk == nil {
		clk = clock.Real{}
	}
	if probe == nil {
		probe = DefaultProbe{}
	}
	if pollEvery <= 0 {
		pollEvery = constants.DefaultActionSampleInterval
	}
	return &ActionTracker{
		clock:       clk,
		probe:       probe,
		pollEvery:   pollEvery,
		focus:       map[string]*models.AppFocus{},
		windowStart: clk.Now(),
	}
}

func (t *ActionTracker) Sample() {
	app := t.probe.ForegroundApp()
	if app == nil {
		return
	}
	key := app.BundleID
	if key == "" {
		key = app.Name
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lastApp != "" && t.lastApp != key {
		t.switches++
	}
	t.lastApp = key
	if cur, ok := t.focus[key]; ok {
		cur.DurationSec += t.pollEvery.Seconds()
	} else {
		t.focus[key] = &models.AppFocus{
			AppName:     app.Name,
			BundleID:    app.BundleID,
			DurationSec: t.pollEvery.Seconds(),
		}
	}
}

func (t *ActionTracker) SnapshotAndReset() models.ActionSummary {
	end := t.clock.Now()
	idle := t.probe.IdleSeconds()
	presence := constants.PresenceActive
	if idle >= 60 {
		presence = constants.PresenceIdle
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	start := t.windowStart
	focus := make([]models.AppFocus, 0, len(t.focus))
	for _, f := range t.focus {
		focus = append(focus, *f)
	}
	summary := models.ActionSummary{
		WindowStart: start,
		WindowEnd:   end,
		Focus:       focus,
		Presence:    presence,
		IdleSec:     idle,
		AppSwitches: t.switches,
	}
	t.focus = map[string]*models.AppFocus{}
	t.switches = 0
	t.windowStart = end
	return summary
}

func ActionSummaryPayload(s models.ActionSummary) map[string]any {
	focus := make([]map[string]any, 0, len(s.Focus))
	for _, f := range s.Focus {
		focus = append(focus, map[string]any{
			"app_name":     f.AppName,
			"bundle_id":    f.BundleID,
			"duration_sec": f.DurationSec,
		})
	}
	return map[string]any{
		"window_start": s.WindowStart.UTC().Format(time.RFC3339),
		"window_end":   s.WindowEnd.UTC().Format(time.RFC3339),
		"focus":        focus,
		"presence":     s.Presence,
		"idle_sec":     s.IdleSec,
		"app_switches": s.AppSwitches,
	}
}
