package network

import "testing"

func TestParseLsofLISTEN(t *testing.T) {
	raw := `COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
ollama  1234 user   5u  IPv4 0xabc      0t0  TCP 127.0.0.1:11434 (LISTEN)
python  9999 user   3u  IPv4 0xdef      0t0  TCP *:8080 (LISTEN)
`
	got := parseLsofLISTEN(raw)
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[0].PID != 1234 || got[0].Port != 11434 || got[0].Addr != "127.0.0.1" {
		t.Fatalf("%+v", got[0])
	}
	if got[1].Addr != "0.0.0.0" || got[1].Port != 8080 {
		t.Fatalf("%+v", got[1])
	}
}

func TestParseLsofLISTENEmpty(t *testing.T) {
	if got := parseLsofLISTEN(""); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}
