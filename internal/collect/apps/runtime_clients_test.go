package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestEngineLocalClientsCorrelate(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 55, Executable: path, Comm: "ollama"},
				{PID: 10, Executable: "/Applications/Cursor.app/Contents/MacOS/Cursor", Comm: "Cursor"},
			}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{PID: 55, Addr: "127.0.0.1", Port: 11434, Protocol: "tcp"}}, nil
		},
		ListLoopback: func() ([]network.LoopbackEstablishedConn, error) {
			return []network.LoopbackEstablishedConn{
				{PID: 10, Comm: "Cursor", LocalAddr: "127.0.0.1", LocalPort: 50000, RemoteAddr: "127.0.0.1", RemotePort: 11434},
				{PID: 55, Comm: "ollama", LocalAddr: "127.0.0.1", LocalPort: 11434, RemoteAddr: "127.0.0.1", RemotePort: 50000},
			}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 1 {
		t.Fatal("expected runtime")
	}
	var runtime *InventoryEntry
	for i := range got {
		if got[i].Identification.Product != nil && got[i].Identification.Product.ID == identity.ProductOllamaID {
			runtime = &got[i]
			break
		}
	}
	if runtime == nil || len(runtime.LocalClients) == 0 {
		t.Fatalf("expected local client on runtime: %+v", got)
	}
	if runtime.LocalClients[0].PID != 10 {
		t.Fatalf("clients=%+v", runtime.LocalClients)
	}
}

func TestPortOnlyConnectDoesNotCreateProduct(t *testing.T) {
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs:  func() ([]process.ProcessInfo, error) { return nil, nil },
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{PID: 1, Addr: "127.0.0.1", Port: 11434}}, nil
		},
		ListLoopback: func() ([]network.LoopbackEstablishedConn, error) {
			return []network.LoopbackEstablishedConn{
				{PID: 2, Comm: "curl", RemotePort: 11434, LocalPort: 1, LocalAddr: "127.0.0.1", RemoteAddr: "127.0.0.1"},
			}, nil
		},
	})
	got, _ := eng.Inventory()
	if len(got) != 0 {
		t.Fatalf("listener/port alone must not create product: %+v", got)
	}
}
