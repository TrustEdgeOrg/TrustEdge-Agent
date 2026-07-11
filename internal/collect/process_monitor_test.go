package collect

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
)

func TestParsePSOutput(t *testing.T) {
	sample := []byte(`  123   1 root     launchd
  456 123 elad     curl
`)
	rows := parsePSOutput(sample)
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2", len(rows))
	}
	if rows[1].PID != 456 || rows[1].PPID != 123 || rows[1].Comm != "curl" {
		t.Fatalf("row1=%+v", rows[1])
	}
}

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
