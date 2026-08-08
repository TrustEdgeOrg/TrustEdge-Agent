package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestParseDockerPSJSONOllama(t *testing.T) {
	raw := []byte(`{"ID":"aafe","Names":"trustedge-dev-ollama","Image":"ollama/ollama:latest","State":"running","Status":"Up 2 hours","Ports":"0.0.0.0:11434->11434/tcp, [::]:11434->11434/tcp"}
{"ID":"redis","Names":"redis","Image":"redis:7-alpine","State":"running","Ports":"0.0.0.0:6379->6379/tcp"}
`)
	got := parseDockerPSJSON(raw)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Image != "ollama/ollama:latest" || got[0].Status != "running" {
		t.Fatalf("%+v", got[0])
	}
	if len(got[0].Ports) < 1 || got[0].Ports[0].HostPort != 11434 {
		t.Fatalf("ports=%+v", got[0].Ports)
	}
}

func TestDockerExecutableHintStripsTag(t *testing.T) {
	if got := dockerExecutableHint("ollama/ollama:latest"); got != "ollama" {
		t.Fatalf("got %q", got)
	}
	if got := dockerExecutableHint("ollama/ollama"); got != "ollama" {
		t.Fatalf("got %q", got)
	}
}

func TestContainerImageMatches(t *testing.T) {
	want := []string{"ollama/ollama"}
	if !containerImageMatches("ollama/ollama:latest", want) {
		t.Fatal("expected match")
	}
	if containerImageMatches("redis:7", want) {
		t.Fatal("redis must not match")
	}
	if containerImageMatches("myorg/fake-ollama:latest", want) {
		t.Fatal("suffix spoof must not match")
	}
}

func TestDockerDiscovererMatchesOllamaImage(t *testing.T) {
	dockerPortsByPath = map[string]DockerContainer{}
	d := &DockerDiscoverer{
		Catalog: identity.DefaultCatalog(),
		ListFn: func(images []string) ([]DockerContainer, error) {
			return []DockerContainer{{
				ID: "aafe", Name: "trustedge-dev-ollama", Image: "ollama/ollama:latest", Status: "running",
				Ports: []DockerPublishedPort{{HostIP: "0.0.0.0", HostPort: 11434, ContainerPort: 11434, Protocol: "tcp"}},
			}, {
				ID: "x", Name: "redis", Image: "redis:7", Status: "running",
			}}, nil
		},
	}
	items, err := d.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d %+v", len(items), items)
	}
	if items[0].PackageManager != "docker" || items[0].PackageIdentifier != "ollama/ollama:latest" {
		t.Fatalf("%+v", items[0])
	}
	if items[0].Executable != "ollama" {
		t.Fatalf("Executable=%q", items[0].Executable)
	}
}

func TestEngineDockerOllamaServing(t *testing.T) {
	dockerPortsByPath = map[string]DockerContainer{}
	path := "docker://trustedge-dev-ollama"
	d := &DockerDiscoverer{
		Catalog: identity.DefaultCatalog(),
		ListFn: func(images []string) ([]DockerContainer, error) {
			return []DockerContainer{{
				ID: "aafe", Name: "trustedge-dev-ollama", Image: "ollama/ollama:latest", Status: "running",
				Ports: []DockerPublishedPort{{HostIP: "0.0.0.0", HostPort: 11434, Protocol: "tcp"}},
			}}, nil
		},
	}
	eng := NewEngine(EngineConfig{
		Discoverer: d,
		Signer:     nil,
		ListProcs:  func() ([]process.ProcessInfo, error) { return nil, nil },
		ListListeners: func() ([]network.ListeningSocket, error) {
			// Host listener owned by Docker Desktop — must not be required for serving.
			return []network.ListeningSocket{{
				PID: 999, Addr: "0.0.0.0", Port: 11434, Protocol: "tcp", Comm: "com.docke",
			}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	e := got[0]
	if e.Identification.Product == nil || e.Identification.Product.ID != identity.ProductOllamaID {
		t.Fatalf("product=%+v", e.Identification.Product)
	}
	if e.Identification.Confidence != identity.ConfidenceHigh {
		t.Fatalf("confidence=%s matched=%v", e.Identification.Confidence, e.Identification.Matched)
	}
	if !e.Installed || !e.Running || !e.Serving {
		t.Fatalf("installed=%v running=%v serving=%v path=%s", e.Installed, e.Running, e.Serving, e.Identity.Path)
	}
	if e.Exposure != ExposureAllInterfaces {
		t.Fatalf("exposure=%s", e.Exposure)
	}
	if e.Identity.Path != path {
		t.Fatalf("path=%s", e.Identity.Path)
	}
	if _, ok := artifactFromEntry(e).Payload["ai_active"]; ok {
		t.Fatal("ai_active forbidden")
	}
}

func TestEnginePortOnlyNotOllama(t *testing.T) {
	dockerPortsByPath = map[string]DockerContainer{}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs:  func() ([]process.ProcessInfo, error) { return nil, nil },
		ListListeners: func() ([]network.ListeningSocket, error) {
			return []network.ListeningSocket{{
				PID: 1, Addr: "127.0.0.1", Port: 11434, Protocol: "tcp", Comm: "python3",
			}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == identity.ProductOllamaID {
			t.Fatal("port alone must not identify ollama")
		}
	}
}

func TestMatcherOllamaAppBundle(t *testing.T) {
	m := identity.NewMatcher(identity.DefaultCatalog())
	res := m.Identify(identity.ApplicationIdentity{
		Path:             "/Applications/Ollama.app",
		Executable:       "Ollama",
		BundleID:         "com.electron.ollama",
		SignatureChecked: true,
		SignatureValid:   true,
	})
	if res.Product == nil || res.Product.ID != identity.ProductOllamaID {
		t.Fatalf("product=%+v", res.Product)
	}
	if res.Confidence != identity.ConfidenceHigh && res.Confidence != identity.ConfidenceMedium {
		t.Fatalf("app bundle should be MEDIUM/HIGH, got %s", res.Confidence)
	}
	if res.Confidence == identity.ConfidenceLow || res.Confidence == identity.ConfidenceUnknown {
		t.Fatalf("too weak: %s", res.Confidence)
	}
}
