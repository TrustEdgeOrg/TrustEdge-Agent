//go:build darwin || linux

package collect

import "testing"

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
