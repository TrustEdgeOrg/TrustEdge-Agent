package network

import (
	"bufio"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// ListeningSocket is a TCP LISTEN socket attributed to a process.
// Port alone must never be used as product identity.
type ListeningSocket struct {
	PID      int
	Comm     string
	Protocol string
	Addr     string
	Port     int
}

// ListListeningSockets is the injectable LISTEN enumerator.
var ListListeningSockets = listListeningSockets

func listListeningSockets() ([]ListeningSocket, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN").Output()
		if err != nil {
			return nil, err
		}
		return parseLsofLISTEN(string(out)), nil
	case "linux":
		out, err := exec.Command("ss", "-ltnp").Output()
		if err != nil {
			return nil, err
		}
		return parseSSLISTEN(string(out)), nil
	default:
		return nil, nil
	}
}

func parseLsofLISTEN(raw string) []ListeningSocket {
	var out []ListeningSocket
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
		if len(fields) < 9 {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || pid <= 0 {
			continue
		}
		comm := basenameComm(fields[0])
		nameIdx := -1
		for i, f := range fields {
			if strings.EqualFold(f, "TCP") || strings.HasPrefix(strings.ToUpper(f), "TCP") {
				nameIdx = i
				break
			}
		}
		if nameIdx < 0 || nameIdx+1 >= len(fields) {
			continue
		}
		endpoint := fields[nameIdx+1]
		// Forms: *:11434  127.0.0.1:11434  [::1]:11434
		endpoint = strings.TrimSuffix(endpoint, " (LISTEN)")
		addr, port := splitHostPort(endpoint)
		if port <= 0 {
			continue
		}
		if addr == "" || addr == "*" {
			addr = "0.0.0.0"
		}
		out = append(out, ListeningSocket{
			PID:      pid,
			Comm:     comm,
			Protocol: "tcp",
			Addr:     addr,
			Port:     port,
		})
	}
	return out
}

func parseSSLISTEN(raw string) []ListeningSocket {
	var out []ListeningSocket
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Netid") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		local := fields[3]
		if strings.EqualFold(fields[0], "LISTEN") && len(fields) > 3 {
			local = fields[3]
		} else if len(fields) >= 4 {
			// ss -ltnp: Netid State Recv-Q Send-Q Local Address:Port Peer Address:Port
			local = fields[3]
		}
		addr, port := splitHostPort(local)
		if port <= 0 {
			continue
		}
		if addr == "" || addr == "*" {
			addr = "0.0.0.0"
		}
		pid := 0
		comm := ""
		joined := strings.Join(fields, " ")
		if m := ssUsersRe.FindStringSubmatch(joined); m != nil {
			comm = m[1]
			pid, _ = strconv.Atoi(m[2])
		}
		if pid <= 0 {
			continue
		}
		out = append(out, ListeningSocket{
			PID:      pid,
			Comm:     basenameComm(comm),
			Protocol: "tcp",
			Addr:     addr,
			Port:     port,
		})
	}
	return out
}
