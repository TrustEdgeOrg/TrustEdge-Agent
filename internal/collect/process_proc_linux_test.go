//go:build linux

package collect

import (
	"encoding/binary"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestProcEventChangeExec(t *testing.T) {
	ev := make([]byte, 24)
	binary.LittleEndian.PutUint32(ev[0:], procEventExec)
	binary.LittleEndian.PutUint32(ev[16:], 4242)

	ch, ok := procEventChange(ev)
	if !ok {
		t.Fatal("expected exec event")
	}
	if ch.Type != constants.TypeProcessStart {
		t.Fatalf("type=%s", ch.Type)
	}
	if intFromAny(ch.Payload["pid"]) != 4242 {
		t.Fatalf("payload=%v", ch.Payload)
	}
}

func TestProcEventChangeExit(t *testing.T) {
	ev := make([]byte, 36)
	binary.LittleEndian.PutUint32(ev[0:], procEventExit)
	binary.LittleEndian.PutUint32(ev[16:], 99)
	binary.LittleEndian.PutUint32(ev[28:], 1)

	ch, ok := procEventChange(ev)
	if !ok {
		t.Fatal("expected exit event")
	}
	if ch.Type != constants.TypeProcessExit {
		t.Fatalf("type=%s", ch.Type)
	}
	if intFromAny(ch.Payload["pid"]) != 99 {
		t.Fatalf("payload=%v", ch.Payload)
	}
}

func TestParseProcStat(t *testing.T) {
	ppid, comm, ok := parseProcStat([]byte("4242 (curl) S 1000 4242 4242 0 -1 ..."))
	if !ok || ppid != 1000 || comm != "curl" {
		t.Fatalf("ppid=%d comm=%q ok=%v", ppid, comm, ok)
	}
}

func TestTruncateCmdline(t *testing.T) {
	short := "curl -I https://example.com"
	if truncateCmdline(short) != short {
		t.Fatal("short cmdline should be unchanged")
	}
	b := make([]byte, maxCmdlineBytes+10)
	for i := range b {
		b[i] = 'a'
	}
	out := truncateCmdline(string(b))
	if len(out) != maxCmdlineBytes+3 || out[len(out)-3:] != "..." {
		t.Fatalf("len=%d suffix=%q", len(out), out[len(out)-3:])
	}
}
