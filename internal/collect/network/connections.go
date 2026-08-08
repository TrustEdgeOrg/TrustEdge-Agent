package network

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EstablishedConn is one TCP ESTABLISHED socket attributed to a process.
type EstablishedConn struct {
	PID            int
	Comm           string
	Protocol       string
	LocalAddr      string
	LocalPort      int
	RemoteAddr     string
	RemotePort     int
	Direction      string
	RemoteHostname string // optional reverse-DNS name
}

func (c EstablishedConn) Fingerprint() string {
	return fmt.Sprintf(
		"%d|%s|%s:%d|%s:%d",
		c.PID,
		c.Protocol,
		c.LocalAddr,
		c.LocalPort,
		c.RemoteAddr,
		c.RemotePort,
	)
}

func (c EstablishedConn) Payload() map[string]any {
	payload := map[string]any{
		"pid":         c.PID,
		"protocol":    c.Protocol,
		"local_addr":  c.LocalAddr,
		"local_port":  c.LocalPort,
		"remote_addr": c.RemoteAddr,
		"remote_ip":   c.RemoteAddr,
		"remote_port": c.RemotePort,
		"direction":   c.Direction,
	}
	if c.Comm != "" {
		payload["comm"] = c.Comm
	}
	host := strings.TrimSpace(c.RemoteHostname)
	if host != "" {
		payload["remote_hostname"] = host
		payload["domain"] = host
	}
	return payload
}

// WithHostname returns a copy with reverse-DNS enrichment when useful.
func (c EstablishedConn) WithHostname() EstablishedConn {
	if c.RemoteHostname != "" || c.RemoteAddr == "" || isNonPublicIP(c.RemoteAddr) {
		return c
	}
	if host := lookupRemoteHostname(c.RemoteAddr); host != "" {
		c.RemoteHostname = host
	}
	return c
}

var (
	hostnameCache   sync.Map // ip -> string (may be empty)
	hostnameTimeout = 150 * time.Millisecond
)

func lookupRemoteHostname(ip string) string {
	if v, ok := hostnameCache.Load(ip); ok {
		return v.(string)
	}
	ctx, cancel := context.WithTimeout(context.Background(), hostnameTimeout)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	host := ""
	if err == nil {
		for _, name := range names {
			name = strings.TrimSuffix(strings.TrimSpace(name), ".")
			if name != "" {
				host = name
				break
			}
		}
	}
	hostnameCache.Store(ip, host)
	return host
}

func isNonPublicIP(addr string) bool {
	ip := net.ParseIP(strings.TrimSpace(addr))
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// listEstablishedConnections returns current TCP ESTABLISHED sockets with PID when available.
func listEstablishedConnections() ([]EstablishedConn, error) {
	switch runtime.GOOS {
	case "darwin":
		return listConnectionsDarwin()
	case "linux":
		return listConnectionsLinux()
	case "windows":
		return listConnectionsWindows()
	default:
		return nil, nil
	}
}

func listConnectionsDarwin() ([]EstablishedConn, error) {
	// lsof -nP: numeric hosts/ports; ESTABLISHED TCP only.
	// Hostnames are enriched separately via reverse DNS (WithHostname).
	out, err := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:ESTABLISHED").Output()
	if err != nil {
		// Non-zero exit with empty output is common when nothing matches.
		if len(out) == 0 {
			return nil, nil
		}
	}
	return parseLsofTCP(string(out)), nil
}

func listConnectionsLinux() ([]EstablishedConn, error) {
	out, err := exec.Command("ss", "-Htanp", "state", "established").Output()
	if err != nil {
		if len(out) == 0 {
			return nil, nil
		}
	}
	return parseSSTCP(string(out)), nil
}

func listConnectionsWindows() ([]EstablishedConn, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsNetstat(string(out)), nil
}

var (
	lsofArrowRe = regexp.MustCompile(`(?i)^(?:TCP|UDP)\s+(.+)$`)
	ssUsersRe   = regexp.MustCompile(`users:\(\("([^"]+)",pid=(\d+)`)
)

func parseLsofTCP(raw string) []EstablishedConn {
	var out []EstablishedConn
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
		comm := fields[0]
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
		if skipAddr(remoteAddr) || remotePort <= 0 || localPort <= 0 {
			continue
		}
		out = append(out, EstablishedConn{
			PID:        pid,
			Comm:       basenameComm(comm),
			Protocol:   "tcp",
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			Direction:  inferDirection(remotePort),
		})
	}
	return out
}

func parseSSTCP(raw string) []EstablishedConn {
	var out []EstablishedConn
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// Recv-Q Send-Q Local Peer [users:(...)]
		if len(fields) < 4 {
			continue
		}
		local := fields[2]
		remote := fields[3]
		localAddr, localPort := splitHostPort(local)
		remoteAddr, remotePort := splitHostPort(remote)
		if skipAddr(remoteAddr) || remotePort <= 0 || localPort <= 0 {
			continue
		}
		pid := 0
		comm := ""
		if len(fields) >= 5 {
			rest := strings.Join(fields[4:], " ")
			if m := ssUsersRe.FindStringSubmatch(rest); m != nil {
				comm = basenameComm(m[1])
				pid, _ = strconv.Atoi(m[2])
			}
		}
		if pid <= 0 {
			continue
		}
		out = append(out, EstablishedConn{
			PID:        pid,
			Comm:       comm,
			Protocol:   "tcp",
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			Direction:  inferDirection(remotePort),
		})
	}
	return out
}

func parseWindowsNetstat(raw string) []EstablishedConn {
	var out []EstablishedConn
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[0], "TCP") {
			continue
		}
		state := strings.ToUpper(fields[3])
		if state != "ESTABLISHED" {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil || pid <= 0 {
			continue
		}
		localAddr, localPort := splitHostPort(fields[1])
		remoteAddr, remotePort := splitHostPort(fields[2])
		if skipAddr(remoteAddr) || remotePort <= 0 || localPort <= 0 {
			continue
		}
		out = append(out, EstablishedConn{
			PID:        pid,
			Protocol:   "tcp",
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			Direction:  inferDirection(remotePort),
		})
	}
	return out
}

func splitEndpoints(name string) (local, remote string, ok bool) {
	name = strings.TrimSpace(name)
	// Strip trailing " (ESTABLISHED)" if present.
	if i := strings.Index(name, " ("); i >= 0 {
		name = name[:i]
	}
	if i := strings.Index(name, "->"); i >= 0 {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+2:]), true
	}
	return "", "", false
}

func splitHostPort(addr string) (host string, port int) {
	addr = strings.TrimSpace(addr)
	if addr == "" || addr == "*.*" || addr == "*:*" {
		return "", 0
	}
	// Bracketed IPv6: [fe80::1]:443
	if strings.HasPrefix(addr, "[") {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return "", 0
		}
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0
		}
		return host, p
	}
	// IPv4 or host with :port (Linux/Windows) or .port (macOS netstat style).
	if host, portStr, err := net.SplitHostPort(addr); err == nil {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0
		}
		return host, p
	}
	if i := strings.LastIndex(addr, "."); i >= 0 {
		p, err := strconv.Atoi(addr[i+1:])
		if err == nil && p > 0 && p <= 65535 {
			return addr[:i], p
		}
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		p, err := strconv.Atoi(addr[i+1:])
		if err == nil && p > 0 && p <= 65535 {
			return addr[:i], p
		}
	}
	return "", 0
}

func skipAddr(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return addr == "" || addr == "*"
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

func inferDirection(remotePort int) string {
	// Heuristic: traffic to common service ports is treated as outbound.
	switch remotePort {
	case 80, 443, 8080, 8443, 53, 22, 25, 465, 587, 993, 995:
		return "outbound"
	default:
		if remotePort > 0 && remotePort < 1024 {
			return "outbound"
		}
		return "established"
	}
}

func basenameComm(comm string) string {
	comm = strings.TrimSpace(comm)
	if i := strings.LastIndexAny(comm, `/\`); i >= 0 {
		comm = comm[i+1:]
	}
	return comm
}
