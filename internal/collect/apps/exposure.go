package apps

import (
	"net"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
)

// ClassifyListenerExposure returns LOOPBACK_ONLY, ALL_INTERFACES, LAN_EXPOSED, or OTHER.
func ClassifyListenerExposure(listeners []ListenerInfo) string {
	if len(listeners) == 0 {
		return ""
	}
	hasLoopback := false
	hasAll := false
	hasLAN := false
	hasOther := false
	for _, l := range listeners {
		addr := strings.TrimSpace(l.Addr)
		switch {
		case addr == "0.0.0.0" || addr == "::" || addr == "*":
			hasAll = true
		case isLoopbackAddr(addr):
			hasLoopback = true
		case isPrivateLANAddr(addr):
			hasLAN = true
		default:
			hasOther = true
		}
	}
	switch {
	case hasAll:
		return ExposureAllInterfaces
	case hasLAN:
		return ExposureLANExposed
	case hasOther:
		return ExposureOther
	case hasLoopback:
		return ExposureLoopbackOnly
	default:
		return ExposureOther
	}
}

func isLoopbackAddr(addr string) bool {
	ip := net.ParseIP(addr)
	if ip != nil {
		return ip.IsLoopback()
	}
	return addr == "localhost"
}

func isPrivateLANAddr(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() && !ip.IsLoopback()
}

func listenersForPIDs(socks []network.ListeningSocket, pids []int) []ListenerInfo {
	want := make(map[int]struct{}, len(pids))
	for _, p := range pids {
		want[p] = struct{}{}
	}
	var out []ListenerInfo
	for _, s := range socks {
		if _, ok := want[s.PID]; !ok {
			continue
		}
		out = append(out, ListenerInfo{
			Addr:     s.Addr,
			Port:     s.Port,
			Protocol: s.Protocol,
		})
	}
	return out
}

func isLocalModelRuntime(entry *InventoryEntry) bool {
	if entry == nil || entry.Identification.Product == nil {
		return false
	}
	return entry.Identification.Product.Category == "local_model_runtime" ||
		string(entry.Identification.Product.Category) == "local_model_runtime"
}
