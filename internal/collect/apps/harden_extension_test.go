package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestHardenFakeFolderNameNotCline(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "cline")
	_ = os.MkdirAll(extDir, 0o755)
	_ = os.WriteFile(filepath.Join(extDir, "package.json"), []byte(`{"name":"cline","publisher":"evil","version":"1.0.0","displayName":"Cline"}`), 0o644)
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: "/Applications/Cursor.app", Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
			SignatureChecked: true, SignatureValid: true,
		}}},
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{RootsByProduct: map[string][]string{identity.ProductCursorID: {root}}},
		},
	})
	got, _ := eng.Inventory()
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == "cline" {
			t.Fatal("fake publisher must not identify as Cline")
		}
	}
}

func TestHardenInstalledDoesNotImplyActive(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "GitHub.copilot-1.0.0")
	_ = os.MkdirAll(extDir, 0o755)
	_ = os.WriteFile(filepath.Join(extDir, "package.json"), []byte(`{"name":"copilot","publisher":"GitHub","version":"1.0.0"}`), 0o644)
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: "/Applications/Cursor.app", Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
			SignatureChecked: true, SignatureValid: true,
		}}},
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 1, Executable: "/Applications/Cursor.app/Contents/MacOS/Cursor", Comm: "Cursor"}}, nil
		},
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{RootsByProduct: map[string][]string{identity.ProductCursorID: {root}}},
		},
	})
	got, _ := eng.Inventory()
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == "github_copilot" {
			if e.Active != nil {
				t.Fatal("installed must not imply active")
			}
			art := artifactFromEntry(e)
			if _, ok := art.Payload["ai_active"]; ok {
				t.Fatal("ai_active forbidden")
			}
			if art.Payload["category"] != "ai_ide_extension" {
				t.Fatalf("category=%v", art.Payload["category"])
			}
		}
	}
}

func TestHardenSameExtensionVSCodeAndCursor(t *testing.T) {
	cursorRoot := t.TempDir()
	codeRoot := t.TempDir()
	for _, root := range []string{cursorRoot, codeRoot} {
		extDir := filepath.Join(root, "Continue.continue-1.0.0")
		_ = os.MkdirAll(extDir, 0o755)
		_ = os.WriteFile(filepath.Join(extDir, "package.json"), []byte(`{"name":"continue","publisher":"Continue","version":"1.0.0"}`), 0o644)
	}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{
			{
				Path: "/Applications/Cursor.app", Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
				SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
				SignatureChecked: true, SignatureValid: true,
			},
			{
				Path: "/Applications/Visual Studio Code.app", Executable: "Code", BundleID: "com.microsoft.VSCode",
				SignatureChecked: true, SignatureValid: true,
			},
		}},
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{RootsByProduct: map[string][]string{
				identity.ProductCursorID: {cursorRoot},
				identity.ProductVSCodeID: {codeRoot},
			}},
		},
	})
	got, _ := eng.Inventory()
	var n int
	hosts := map[string]bool{}
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == "continue" {
			n++
			hosts[e.HostIDEProductID] = true
		}
	}
	if n != 2 {
		t.Fatalf("want 2 continue installs, got %d", n)
	}
	if !hosts[identity.ProductCursorID] || !hosts[identity.ProductVSCodeID] {
		t.Fatalf("hosts=%v", hosts)
	}
}

func TestHardenNoHomeRecursion(t *testing.T) {
	p := &VSCodeCompatibleExtensionProvider{}
	roots := p.extensionRoots(HostIDEIdentity{ProductID: identity.ProductCursorID})
	for _, r := range roots {
		base := filepath.Base(r)
		if base != "extensions" {
			t.Fatalf("unexpected root %s", r)
		}
	}
}

func TestFolderExtensionIDHint(t *testing.T) {
	if got := folderExtensionIDHint("saoudrizwan.claude-dev-3.8.0"); got != "saoudrizwan.claude-dev" {
		t.Fatalf("got %q", got)
	}
}
