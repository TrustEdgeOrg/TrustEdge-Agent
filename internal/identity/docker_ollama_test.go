package identity

import "testing"

func TestOllamaDockerImageHighConfidence(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "docker://trustedge-dev-ollama",
		Executable:        "ollama",
		PackageManager:    "docker",
		PackageIdentifier: "ollama/ollama:latest",
	})
	if res.Product == nil || res.Product.ID != ProductOllamaID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence != ConfidenceHigh {
		t.Fatalf("docker image match want HIGH, got %s matched=%v", res.Confidence, res.Matched)
	}
	if !hasEvidence(res.Matched, EvidenceDockerImage) {
		t.Fatal("expected docker_image evidence")
	}
}

func TestOllamaAppBundleMediumOrHigh(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:             "/Applications/Ollama.app",
		Executable:       "Ollama",
		BundleID:         "com.electron.ollama",
		SignatureChecked: true,
		SignatureValid:   true,
	})
	if res.Product == nil || res.Product.ID != ProductOllamaID {
		t.Fatalf("product=%v", res.Product)
	}
	if res.Confidence != ConfidenceHigh && res.Confidence != ConfidenceMedium {
		t.Fatalf("got %s", res.Confidence)
	}
}

func TestFakeDockerImageNotOllama(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "docker://evil",
		Executable:        "ollama",
		PackageManager:    "docker",
		PackageIdentifier: "evil/fake-ollama:latest",
	})
	// May still candidate-match via ExecutableNames "ollama", but must not be HIGH via docker_image.
	if res.Product != nil && res.Product.ID == ProductOllamaID {
		if hasEvidence(res.Matched, EvidenceDockerImage) {
			t.Fatal("fake image must not match docker_image")
		}
		if res.Confidence == ConfidenceHigh || res.Confidence == ConfidenceVerified {
			t.Fatalf("fake docker image too strong: %s", res.Confidence)
		}
	}
}
