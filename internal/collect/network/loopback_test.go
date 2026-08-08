package network

import "testing"

func TestParseLsofLoopbackEstablished(t *testing.T) {
	raw := `COMMAND  PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
Cursor  10 user   1u  IPv4 0x1      0t0  TCP 127.0.0.1:54321->127.0.0.1:11434 (ESTABLISHED)
ollama  55 user   2u  IPv4 0x2      0t0  TCP 127.0.0.1:11434->127.0.0.1:54321 (ESTABLISHED)
Safari  99 user   3u  IPv4 0x3      0t0  TCP 10.0.0.2:443->1.2.3.4:443 (ESTABLISHED)
`
	got := parseLsofLoopbackEstablished(raw)
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
}
