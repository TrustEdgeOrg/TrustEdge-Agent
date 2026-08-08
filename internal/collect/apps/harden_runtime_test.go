package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestHardenSpoofOllamaNameNotVerified(t *testing.T) {
	m := identity.NewMatcher(identity.DefaultCatalog())
	res := m.Identify(identity.ApplicationIdentity{
		Executable: "ollama", Path: "/tmp/ollama",
	})
	if res.Confidence == identity.ConfidenceVerified || res.Confidence == identity.ConfidenceHigh {
		t.Fatalf("spoof too strong: %s", res.Confidence)
	}
	if res.Product.Category != identity.ProductCategoryLocalModelRuntime {
		t.Fatal("must stay local_model_runtime")
	}
	if res.Product.Category == identity.ProductCategoryCLIAgent {
		t.Fatal("must never become cli_agent")
	}
}

func TestHardenGenericHTTPSamePort(t *testing.T) {
	path := "/opt/homebrew/bin/ollama"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: path, Executable: "ollama", ExecutablePath: path, ResolvedPath: path,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 55, Executable: path, Comm: "ollama"},
				{PID: 80, Executable: "/usr/bin/python3", Comm: "python3"},
			}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{
				{PID: 80, Addr: "127.0.0.1", Port: 11434, Protocol: "tcp", Comm: "python3"},
			}, nil
		},
	})
	got, _ := eng.Inventory()
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == identity.ProductOllamaID && e.Serving {
			t.Fatal("python owning port must not mark ollama serving")
		}
	}
}

func TestHardenMultipleRuntimes(t *testing.T) {
	o := "/opt/homebrew/bin/ollama"
	l := "/usr/local/bin/llama-server"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{
			{Path: o, Executable: "ollama", ExecutablePath: o, ResolvedPath: o},
			{Path: l, Executable: "llama-server", ExecutablePath: l, ResolvedPath: l},
		}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 1, Executable: o, Comm: "ollama"},
				{PID: 2, Executable: l, Comm: "llama-server"},
			}, nil
		},
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{
				{PID: 1, Addr: "127.0.0.1", Port: 11434},
				{PID: 2, Addr: "127.0.0.1", Port: 8080},
			}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 runtimes, got %d", len(got))
	}
	for _, e := range got {
		if e.Identification.Product.Category != identity.ProductCategoryLocalModelRuntime {
			t.Fatalf("category=%s", e.Identification.Product.Category)
		}
		if !e.Serving {
			t.Fatalf("expected serving %+v", e)
		}
		if _, ok := artifactFromEntry(e).Payload["ai_active"]; ok {
			t.Fatal("ai_active forbidden")
		}
	}
}

func TestHardenNoBrewOnInventoryHotPath(t *testing.T) {
	// Provenance is FS-only; inventory must not shell out to brew.
	id := identity.ApplicationIdentity{
		ResolvedPath: "/opt/homebrew/Cellar/ollama/0.9.0/bin/ollama",
	}
	ApplyPackageProvenance(&id)
	if id.PackageManager != "homebrew" || id.PackageIdentifier != "ollama" {
		t.Fatalf("%+v", id)
	}
}
