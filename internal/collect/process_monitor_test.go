package collect

import (
	"strings"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestProcessMonitorBaselineThenStart(t *testing.T) {
	orig := listProcesses
	defer func() { listProcesses = orig }()

	snap1 := []processRow{{PID: 1, PPID: 0, User: "root", Comm: "launchd", Executable: "launchd"}}
	snap2 := append(snap1, processRow{PID: 99, PPID: 1, User: "elad", Comm: "curl", Executable: "curl"})

	calls := 0
	listProcesses = func() ([]processRow, error) {
		calls++
		if calls == 1 {
			return snap1, nil
		}
		return snap2, nil
	}

	m := NewProcessMonitor(nil)
	if got := m.Poll(); len(got) != 0 {
		t.Fatalf("baseline poll=%v want empty", got)
	}
	changes := m.Poll()
	if len(changes) != 1 {
		t.Fatalf("changes=%v want 1 start", changes)
	}
	if changes[0].Type != constants.TypeProcessStart {
		t.Fatalf("type=%s", changes[0].Type)
	}
	if changes[0].Payload["comm"] != "curl" {
		t.Fatalf("payload=%v", changes[0].Payload)
	}
}

func TestProcessMonitorObserveDedup(t *testing.T) {
	m := NewProcessMonitor(nil)
	start := ProcessChange{
		Type: constants.TypeProcessStart,
		Payload: map[string]any{
			"pid":     10,
			"ppid":    1,
			"comm":    "curl",
			"cmdline": "curl https://example.com",
		},
	}
	if !m.Observe(start) {
		t.Fatal("first observe should post")
	}
	if m.Observe(start) {
		t.Fatal("duplicate start should not post")
	}

	exit := ProcessChange{
		Type: constants.TypeProcessExit,
		Payload: map[string]any{
			"pid": 10,
		},
	}
	if !m.Observe(exit) {
		t.Fatal("exit should post")
	}
	if exit.Payload["cmdline"] != "curl https://example.com" {
		t.Fatalf("exit should inherit cmdline, got %v", exit.Payload["cmdline"])
	}
	if exit.Payload["comm"] != "curl" {
		t.Fatalf("exit should inherit comm, got %v", exit.Payload["comm"])
	}
}

func TestTruncateCmdlineShared(t *testing.T) {
	short := "echo hi"
	if truncateCmdline(short) != short {
		t.Fatal("short unchanged")
	}
	b := make([]byte, maxCmdlineBytes+5)
	for i := range b {
		b[i] = 'x'
	}
	out := truncateCmdline(string(b))
	if len(out) != maxCmdlineBytes+3 || !strings.HasSuffix(out, "...") {
		t.Fatalf("got len=%d", len(out))
	}
}

func TestJoinNullSeparatedCmdline(t *testing.T) {
	got := joinNullSeparatedCmdline([]byte("curl\x00-I\x00https://example.com\x00"))
	if got != "curl -I https://example.com" {
		t.Fatalf("got=%q", got)
	}
	if joinNullSeparatedCmdline(nil) != "" {
		t.Fatal("nil should be empty")
	}
}

func TestProcessMonitorCapDoesNotAbsorb(t *testing.T) {
	orig := listProcesses
	defer func() { listProcesses = orig }()

	base := []processRow{{PID: 1, PPID: 0, Comm: "init", Executable: "init"}}
	var snap []processRow
	for i := 2; i <= 105; i++ {
		snap = append(snap, processRow{PID: i, PPID: 1, Comm: "p", Executable: "p"})
	}

	calls := 0
	listProcesses = func() ([]processRow, error) {
		calls++
		if calls == 1 {
			return base, nil
		}
		return append(append([]processRow{}, base...), snap...), nil
	}

	m := NewProcessMonitor(nil)
	_ = m.Poll()
	first := m.Poll()
	if len(first) != maxProcessEventsPerPoll {
		t.Fatalf("first=%d want cap %d", len(first), maxProcessEventsPerPoll)
	}
	second := m.Poll()
	if len(second) == 0 {
		t.Fatal("expected remaining starts on next poll")
	}
}
