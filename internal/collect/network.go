package collect

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

func NetworkSummary() models.NetworkSummary {
	return NewCollector(nil, nil, "", constants.PublicIPLookupURL).NetworkSummary()
}

func NetworkSummaryPayload() map[string]any {
	return NewCollector(nil, nil, "", constants.PublicIPLookupURL).NetworkSummaryPayload()
}

func fetchPublicIP(client *http.Client, url string) string {
	if url == "" {
		return ""
	}
	if client == nil {
		client = &http.Client{Timeout: constants.PublicIPLookupTimeout}
	}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var body struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.IP)
}

func networkType() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return constants.NetworkTypeUnknown
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}
		if strings.HasPrefix(name, "en") || strings.Contains(name, "eth") {
			// macOS: en0/en1 are usually Wi-Fi; other en* interfaces are treated as ethernet.
			if runtime.GOOS == "darwin" && (name == "en0" || name == "en1") {
				return constants.NetworkTypeWiFi
			}
			return constants.NetworkTypeEthernet
		}
		if strings.HasPrefix(name, "wl") || strings.Contains(name, "wifi") {
			return constants.NetworkTypeWiFi
		}
	}
	return constants.NetworkTypeUnknown
}

func portStats() (listening, established int, top []models.PortCount) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return 0, 0, nil
	}
	out, err := exec.Command("netstat", "-an").Output()
	if err != nil {
		return 0, 0, nil
	}
	counts := map[int]int{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		proto := strings.ToLower(fields[0])
		if !strings.HasPrefix(proto, "tcp") && !strings.HasPrefix(proto, "udp") {
			continue
		}
		state := ""
		if len(fields) >= 6 {
			state = strings.ToUpper(fields[len(fields)-1])
		}
		switch state {
		case "LISTEN", "LISTENING":
			listening++
		case "ESTABLISHED":
			established++
			if port := remotePort(fields[4]); port > 0 {
				counts[port]++
			}
		}
	}
	type kv struct {
		port  int
		count int
	}
	var list []kv
	for p, c := range counts {
		list = append(list, kv{p, c})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].port < list[j].port
		}
		return list[i].count > list[j].count
	})
	if len(list) > 5 {
		list = list[:5]
	}
	for _, item := range list {
		top = append(top, models.PortCount{Port: item.port, Count: item.count})
	}
	return listening, established, top
}

func remotePort(addr string) int {
	// netstat -an uses 1.2.3.4.443 on macOS and 1.2.3.4:443 on Linux.
	addr = strings.TrimSpace(addr)
	if addr == "" || addr == "*.*" {
		return 0
	}
	var portStr string
	if i := strings.LastIndex(addr, "."); i >= 0 {
		portStr = addr[i+1:]
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		portStr = addr[i+1:]
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}
