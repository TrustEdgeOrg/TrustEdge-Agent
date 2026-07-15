package collect

import (
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
)

type stubProbe struct {
	app  *ForegroundInfo
	idle float64
}

func (s stubProbe) OSVersion() string             { return "test" }
func (s stubProbe) ForegroundApp() *ForegroundInfo { return s.app }
func (s stubProbe) IdleSeconds() float64           { return s.idle }

type sequenceProbe struct {
	apps []*ForegroundInfo
	idx  int
	idle float64
}

func (s *sequenceProbe) OSVersion() string { return "test" }
func (s *sequenceProbe) ForegroundApp() *ForegroundInfo {
	if len(s.apps) == 0 {
		return nil
	}
	if s.idx >= len(s.apps) {
		return s.apps[len(s.apps)-1]
	}
	app := s.apps[s.idx]
	s.idx++
	return app
}

func (s *sequenceProbe) IdleSeconds() float64 { return s.idle }

func TestActionTrackerAccumulatesSamplesAcrossWindow(t *testing.T) {
	probe := &sequenceProbe{
		apps: []*ForegroundInfo{
			{Name: "Mail", BundleID: "com.mail"},
			{Name: "Mail", BundleID: "com.mail"},
			{Name: "Safari", BundleID: "com.safari"},
			{Name: "Safari", BundleID: "com.safari"},
		},
	}
	tracker := NewActionTracker(clock.Real{}, probe, 5*time.Second)

	for i := 0; i < 4; i++ {
		tracker.Sample()
	}
	summary := tracker.SnapshotAndReset()
	if summary.AppSwitches != 1 {
		t.Fatalf("switches=%d want 1", summary.AppSwitches)
	}
	if len(summary.Focus) != 2 {
		t.Fatalf("focus apps=%d want 2", len(summary.Focus))
	}
	byID := map[string]float64{}
	for _, f := range summary.Focus {
		byID[f.BundleID] = f.DurationSec
	}
	if byID["com.mail"] != 10 || byID["com.safari"] != 10 {
		t.Fatalf("durations=%v want mail=10 safari=10", byID)
	}
}

func TestActionTrackerResetClearsWindow(t *testing.T) {
	probe := stubProbe{app: &ForegroundInfo{Name: "Mail", BundleID: "com.mail"}}
	tracker := NewActionTracker(clock.Real{}, probe, time.Second)
	tracker.Sample()
	_ = tracker.SnapshotAndReset()
	summary := tracker.SnapshotAndReset()
	if len(summary.Focus) != 0 || summary.AppSwitches != 0 {
		t.Fatalf("expected empty window after reset, got %+v", summary)
	}
}
