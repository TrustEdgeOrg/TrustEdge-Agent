package network

import (
	"bufio"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// LoopbackEstablishedConn is an ESTABLISHED TCP socket on loopback.
// Used only for local AI client↔runtime correlation — not public network_connection telemetry.
type LoopbackEstablishedConn struct {
	PID        int
	Comm       string
	LocalAddr  string
	LocalPort  int
	RemoteAddr string
	RemotePort int
}

// ListLoopbackEstablished is injectable for tests.
var ListLoopbackEstablished = listLoopbackEstablished

func listLoopbackEstablished() ([]LoopbackEstablishedConn, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:ESTABLISHED").Output()
		if err != nil {
			return nil, err
		}
		return parseLsofLoopbackEstablished(string(out)), nil
	default:
		return nil, nil
	}
}

func parseLsofLoopbackEstablished(raw string) []LoopbackEstablishedConn {
	var out []LoopbackEstablishedConn
	scanner := bufio.NewScanner(strings.NewReader(raw))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if strings.HasPrefix(strings.ToUpper(line), "COMMAND") {
				continue
			}
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || pid <= 0 {
			continue
		}
		comm := basenameComm(fields[0])
		tcpIdx := -1
		for i, f := range fields {
			upper := strings.ToUpper(f)
			if upper == "TCP" || strings.HasPrefix(upper, "TCP") {
				tcpIdx = i
				break
			}
		}
		if tcpIdx < 0 || tcpIdx+1 >= len(fields) {
			continue
		}
		nameField := strings.Join(fields[tcpIdx:], " ")
		m := lsofArrowRe.FindStringSubmatch(nameField)
		if m == nil {
			continue
		}
		local, remote, ok := splitEndpoints(m[1])
		if !ok {
			continue
		}
		localAddr, localPort := splitHostPort(local)
		remoteAddr, remotePort := splitHostPort(remote)
		if localPort <= 0 || remotePort <= 0 {
			continue
		}
		if !isLoopbackIP(localAddr) || !isLoopbackIP(remoteAddr) {
			continue
		}
		out = append(out, LoopbackEstablishedConn{
			PID:        pid,
			Comm:       comm,
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
		})
	}
	return out
}

func isLoopbackIP(addr string) bool {
	a := strings.TrimSpace(addr)
	return a == "127.0.0.1" || a == "::1" || a == "localhost"
}
