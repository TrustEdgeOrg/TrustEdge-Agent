package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestEngineOllamaRunningAndExit(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	list := []process.ProcessInfo{{PID: 44, Executable: path, Comm: "ollama", StartTimeUnixNano: 1}}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return list, nil },
	})
	first, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || !first[0].Running || !first[0].Installed {
		t.Fatalf("%+v", first)
	}
	list = nil
	eng.NoteExit(44)
	second, _ := eng.Inventory()
	if second[0].Running {
		t.Fatal("exit must clear running")
	}
}

func TestEngineOllamaPIDReuse(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	list := []process.ProcessInfo{{PID: 9, Executable: path, Comm: "ollama", StartTimeUnixNano: 10}}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return list, nil },
	})
	_, _ = eng.Inventory()
	list = []process.ProcessInfo{{PID: 9, Executable: "/usr/bin/true", Comm: "true", StartTimeUnixNano: 99}}
	got, _ := eng.Inventory()
	if got[0].Running {
		t.Fatal("PID reuse must not keep runtime running")
	}
}

func TestEnginePythonNodeNotRuntime(t *testing.T) {
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 1, Executable: "/usr/bin/python3", Comm: "python3"},
				{PID: 2, Executable: "/usr/local/bin/node", Comm: "node"},
			}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("generic interpreters must not match: %+v", got)
	}
}
