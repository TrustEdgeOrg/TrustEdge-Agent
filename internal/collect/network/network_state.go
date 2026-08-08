package network

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

func linkFingerprint() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(ifaces))
	byName := map[string][]string{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}
		names = append(names, iface.Name)
		parts := make([]string, 0, len(addrs))
		for _, a := range addrs {
			parts = append(parts, a.String())
		}
		sort.Strings(parts)
		byName[iface.Name] = parts
	}
	sort.Strings(names)
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(strings.Join(byName[name], ","))
	}
	return b.String()
}

func summaryFingerprint(payload map[string]any) string {
	ports := portFingerprint(payload["top_remote_ports"])
	return fmt.Sprintf(
		"ip:%s|type:%s|listen:%v|est:%v|ports:%s",
		payload["public_ip"],
		payload["network_type"],
		payload["listening_count"],
		payload["established_count"],
		ports,
	)
}

func portFingerprint(v any) string {
	list, ok := v.([]map[string]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(list))
	for _, item := range list {
		parts = append(parts, fmt.Sprintf("%v:%v", item["port"], item["count"]))
	}
	return strings.Join(parts, ",")
}
