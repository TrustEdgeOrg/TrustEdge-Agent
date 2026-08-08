package apps

import (
	"path/filepath"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func (e *Engine) attachLocalClients(byPath map[string]*InventoryEntry, conns []network.LoopbackEstablishedConn) {
	if len(conns) == 0 {
		return
	}
	// port -> runtime entries that listen on it
	byPort := make(map[int][]*InventoryEntry)
	seen := make(map[*InventoryEntry]struct{})
	for _, eptr := range byPath {
		if _, ok := seen[eptr]; ok {
			continue
		}
		seen[eptr] = struct{}{}
		if !isLocalModelRuntime(eptr) || !eptr.Serving {
			continue
		}
		for _, l := range eptr.Listeners {
			byPort[l.Port] = append(byPort[l.Port], eptr)
		}
	}
	if len(byPort) == 0 {
		return
	}

	serverPIDs := make(map[int]struct{})
	for _, eptr := range byPath {
		if !isLocalModelRuntime(eptr) {
			continue
		}
		for _, pid := range eptr.PIDs {
			serverPIDs[pid] = struct{}{}
		}
	}

	for _, c := range conns {
		if _, isServer := serverPIDs[c.PID]; isServer {
			continue // skip the server-side half of the connection
		}
		targets := byPort[c.RemotePort]
		if len(targets) == 0 {
			continue
		}
		client := LocalClientInfo{
			PID:        c.PID,
			Executable: firstNonEmpty(c.Comm, filepath.Base(c.Comm)),
		}
		if e != nil && e.matcher != nil {
			res := e.matcher.Identify(identity.ApplicationIdentity{
				Executable:     c.Comm,
				ExecutablePath: c.Comm,
				Path:           c.Comm,
			})
			if res.Product != nil {
				client.ProductID = res.Product.ID
			}
		}
		for _, eptr := range targets {
			if clientPIDIn(eptr.LocalClients, client.PID) {
				continue
			}
			eptr.LocalClients = append(eptr.LocalClients, client)
			if !hasEvidence(eptr.Identification.Matched, identity.EvidenceLocalClient) {
				eptr.Identification.Matched = append(eptr.Identification.Matched, identity.EvidenceLocalClient)
			}
		}
	}
}

func clientPIDIn(clients []LocalClientInfo, pid int) bool {
	for _, c := range clients {
		if c.PID == pid {
			return true
		}
	}
	return false
}
