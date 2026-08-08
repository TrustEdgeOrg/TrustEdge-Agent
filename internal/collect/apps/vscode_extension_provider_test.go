package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestVSCodeProviderSupports(t *testing.T) {
	p := &VSCodeCompatibleExtensionProvider{}
	if !p.Supports(HostIDEIdentity{ProductID: identity.ProductCursorID}) {
		t.Fatal("cursor")
	}
	if !p.Supports(HostIDEIdentity{ProductID: identity.ProductVSCodeID}) {
		t.Fatal("vscode")
	}
	if p.Supports(HostIDEIdentity{ProductID: "ollama"}) {
		t.Fatal("ollama must not be supported")
	}
}

func TestVSCodeProviderDiscoversBoundedRoots(t *testing.T) {
	root := t.TempDir()
	extRoot := filepath.Join(root, "extensions")
	if err := os.MkdirAll(filepath.Join(extRoot, "publisher.ext-1.0.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tmp", "publisher.ext-9.9.9"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &VSCodeCompatibleExtensionProvider{
		RootsByProduct: map[string][]string{
			identity.ProductCursorID: {extRoot},
		},
	}
	got, err := p.DiscoverInstallations(HostIDEIdentity{
		ProductID: identity.ProductCursorID,
		Path:      "/Applications/Cursor.app",
		Family:    "vscode_compatible",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[0].FolderName != "publisher.ext-1.0.0" {
		t.Fatalf("%+v", got[0])
	}
	if got[0].PackageManager != identity.PackageManagerCursorExtension {
		t.Fatalf("pkg=%s", got[0].PackageManager)
	}
}

func TestVSCodeProviderMissingRoot(t *testing.T) {
	p := &VSCodeCompatibleExtensionProvider{
		RootsByProduct: map[string][]string{
			identity.ProductVSCodeID: {filepath.Join(t.TempDir(), "missing")},
		},
	}
	got, err := p.DiscoverInstallations(HostIDEIdentity{ProductID: identity.ProductVSCodeID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d", len(got))
	}
}
