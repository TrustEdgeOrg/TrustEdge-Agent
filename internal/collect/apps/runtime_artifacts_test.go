package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestScanRuntimeArtifactsOllamaLayout(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".ollama")
	models := filepath.Join(root, "models")
	if err := os.MkdirAll(models, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(models, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(models, "b"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &identity.KnownAIProduct{
		ID:                identity.ProductOllamaID,
		ArtifactPathHints: []string{"~/.ollama"},
	}
	res := ScanRuntimeArtifacts(p, home)
	if !res.Found || res.ModelsAvailable < 2 {
		t.Fatalf("%+v", res)
	}
}

func TestScanGGUFHeaderBounded(t *testing.T) {
	dir := t.TempDir()
	gguf := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(gguf, append([]byte("GGUF"), make([]byte, 100)...), 0o644); err != nil {
		t.Fatal(err)
	}
	huge := filepath.Join(dir, "huge.gguf")
	if err := os.WriteFile(huge, append([]byte("GGUF"), make([]byte, 1024)...), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &identity.KnownAIProduct{ArtifactPathHints: []string{dir}}
	res := ScanRuntimeArtifacts(p, "")
	if res.ModelFormat != "GGUF" || res.ModelsAvailable < 2 {
		t.Fatalf("%+v", res)
	}
}

func TestGGUFAloneNotRuntimeProduct(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.gguf"), []byte("GGUF...."), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs:  func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("GGUF without runtime must not create inventory: %+v", got)
	}
	res := ScanRuntimeArtifacts(nil, dir)
	if res.Found {
		t.Fatal("nil product must not scan")
	}
}

func TestInaccessibleArtifactDir(t *testing.T) {
	p := &identity.KnownAIProduct{ArtifactPathHints: []string{"/no/such/ollama/dir"}}
	res := ScanRuntimeArtifacts(p, "")
	if res.Found {
		t.Fatal("missing dir must not Found")
	}
}
