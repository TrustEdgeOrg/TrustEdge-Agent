package network

import (
	"reflect"
	"testing"
)

func TestParseLsofTCP(t *testing.T) {
	raw := `COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
curl    12345 elad    5u  IPv4 0t0  TCP 192.168.1.10:54321->203.0.113.9:443 (ESTABLISHED)
Chrome  99999 elad   40u  IPv4 0t0  TCP 127.0.0.1:5555->127.0.0.1:443 (ESTABLISHED)
`
	got := parseLsofTCP(raw)
	if len(got) != 1 {
		t.Fatalf("got %d conns, want 1 (loopback filtered)", len(got))
	}
	c := got[0]
	if c.PID != 12345 || c.Comm != "curl" || c.RemoteAddr != "203.0.113.9" || c.RemotePort != 443 {
		t.Fatalf("unexpected conn: %+v", c)
	}
	if c.LocalPort != 54321 || c.Direction != "outbound" {
		t.Fatalf("unexpected local/direction: %+v", c)
	}
}

func TestParseSSTCP(t *testing.T) {
	raw := `0 0 192.168.1.10:54321 203.0.113.9:443 users:(("curl",pid=12345,fd=5))
0 0 127.0.0.1:9 127.0.0.1:80 users:(("local",pid=1,fd=1))
`
	got := parseSSTCP(raw)
	if len(got) != 1 {
		t.Fatalf("got %d conns, want 1", len(got))
	}
	if got[0].PID != 12345 || got[0].RemotePort != 443 || got[0].Comm != "curl" {
		t.Fatalf("unexpected: %+v", got[0])
	}
}

func TestParseWindowsNetstat(t *testing.T) {
	raw := `  Proto  Local Address          Foreign Address        State           PID
  TCP    192.168.1.10:54321     203.0.113.9:443        ESTABLISHED     12345
  TCP    0.0.0.0:80             0.0.0.0:0              LISTENING       4
`
	got := parseWindowsNetstat(raw)
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].PID != 12345 || got[0].RemoteAddr != "203.0.113.9" {
		t.Fatalf("unexpected: %+v", got[0])
	}
}

func TestConnectionMonitorEmitsNewOnly(t *testing.T) {
	calls := 0
	m := NewConnectionMonitor(nil)
	m.List = func() ([]EstablishedConn, error) {
		calls++
		base := EstablishedConn{
			PID: 1, Comm: "a", Protocol: "tcp",
			LocalAddr: "10.0.0.1", LocalPort: 1000,
			RemoteAddr: "1.1.1.1", RemotePort: 443, Direction: "outbound",
		}
		if calls == 1 {
			return []EstablishedConn{base}, nil
		}
		extra := base
		extra.RemoteAddr = "8.8.8.8"
		extra.LocalPort = 1001
		return []EstablishedConn{base, extra}, nil
	}

	if changes := m.Poll(); changes != nil {
		t.Fatalf("first poll should seed silently, got %#v", changes)
	}
	changes := m.Poll()
	if len(changes) != 1 {
		t.Fatalf("second poll want 1 new, got %d", len(changes))
	}
	if changes[0].RemoteAddr != "8.8.8.8" {
		t.Fatalf("conn=%+v", changes[0])
	}
	if changes = m.Poll(); len(changes) != 0 {
		t.Fatalf("third poll want 0, got %d", len(changes))
	}
}

func TestEstablishedConnPayload(t *testing.T) {
	c := EstablishedConn{
		PID: 7, Comm: "curl", Protocol: "tcp",
		LocalAddr: "10.0.0.2", LocalPort: 9,
		RemoteAddr: "9.9.9.9", RemotePort: 443, Direction: "outbound",
		RemoteHostname: "dns.google",
	}
	got := c.Payload()
	wantKeys := []string{"pid", "comm", "protocol", "local_addr", "local_port", "remote_addr", "remote_ip", "remote_port", "direction", "remote_hostname", "domain"}
	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Fatalf("missing key %s in %v", k, got)
		}
	}
	if !reflect.DeepEqual(got["pid"], 7) {
		t.Fatalf("pid=%v", got["pid"])
	}
	if got["remote_hostname"] != "dns.google" || got["domain"] != "dns.google" {
		t.Fatalf("hostname fields=%v", got)
	}
}

func TestWithHostnameSkipsPrivate(t *testing.T) {
	c := EstablishedConn{RemoteAddr: "10.0.0.1", RemotePort: 443}
	got := c.WithHostname()
	if got.RemoteHostname != "" {
		t.Fatalf("expected no hostname for private IP, got %q", got.RemoteHostname)
	}
}
