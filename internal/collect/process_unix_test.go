//go:build darwin

package collect

import "testing"

func TestParsePSOutput(t *testing.T) {
	sample := []byte(`  123   1 root     /sbin/launchd
  456 123 elad     /usr/bin/curl https://example.com
`)
	rows := parsePSOutput(sample)
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2", len(rows))
	}
	if rows[1].PID != 456 || rows[1].PPID != 123 || rows[1].Comm != "curl" {
		t.Fatalf("row1=%+v", rows[1])
	}
	if rows[1].Cmdline != "/usr/bin/curl https://example.com" {
		t.Fatalf("cmdline=%q", rows[1].Cmdline)
	}
	if rows[1].Executable != "/usr/bin/curl" {
		t.Fatalf("executable=%q", rows[1].Executable)
	}
}
