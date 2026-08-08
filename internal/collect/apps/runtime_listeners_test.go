package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestClassifyListenerExposure(t *testing.T) {
	if got := ClassifyListenerExposure([]ListenerInfo{{Addr: "127.0.0.1", Port: 1}}); got != ExposureLoopbackOnly {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyListenerExposure([]ListenerInfo{{Addr: "0.0.0.0", Port: 1}}); got != ExposureAllInterfaces {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyListenerExposure([]ListenerInfo{{Addr: "10.0.0.5", Port: 1}}); got != ExposureLANExposed {
		t.Fatalf("got %s", got)
	}
}

func TestEngineOllamaServingLoopback(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 55, Executable: path, Comm: "ollama"}}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{
				PID: 55, Addr: "127.0.0.1", Port: 11434, Protocol: "tcp", Comm: "ollama",
			}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Serving || got[0].Exposure != ExposureLoopbackOnly {
		t.Fatalf("%+v", got[0])
	}
}

func TestEngineUnrelatedListenerSamePortNotServing(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 55, Executable: path, Comm: "ollama"}}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			// Different PID owns the port — must not attribute to Ollama.
			return []network.ListeningSocket{{
				PID: 999, Addr: "127.0.0.1", Port: 11434, Protocol: "tcp", Comm: "python",
			}}, nil
		},
	})
	got, _ := eng.Inventory()
	if got[0].Serving {
		t.Fatal("port alone must not mark serving without PID match")
	}
}

func TestEngineOllamaChangedPortStillServing(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 55, Executable: path, Comm: "ollama"}}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{
				PID: 55, Addr: "127.0.0.1", Port: 9999, Protocol: "tcp",
			}}, nil
		},
	})
	got, _ := eng.Inventory()
	if !got[0].Serving || got[0].Listeners[0].Port != 9999 {
		t.Fatalf("%+v", got[0])
	}
}

func TestEngineOllamaAllInterfacesExposure(t *testing.T) {
	path := "/usr/local/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 7, Executable: path, Comm: "ollama"}}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{PID: 7, Addr: "0.0.0.0", Port: 11434, Protocol: "tcp"}}, nil
		},
	})
	got, _ := eng.Inventory()
	if got[0].Exposure != ExposureAllInterfaces {
		t.Fatalf("exposure=%s", got[0].Exposure)
	}
}
