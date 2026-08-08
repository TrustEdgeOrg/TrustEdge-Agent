package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestEngineDiscoversClineExtensionUnderCursor(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "saoudrizwan.claude-dev-3.8.0")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{"name":"claude-dev","publisher":"saoudrizwan","version":"3.8.0","displayName":"Cline"}`
	if err := os.WriteFile(filepath.Join(extDir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	cursorPath := "/Applications/Cursor.app"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: cursorPath, Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
			SignatureChecked: true, SignatureValid: true,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{
				RootsByProduct: map[string][]string{identity.ProductCursorID: {root}},
			},
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == "cline" {
			found = true
			if e.HostIDEProductID != identity.ProductCursorID {
				t.Fatalf("host=%s", e.HostIDEProductID)
			}
			if e.ExtensionID != "saoudrizwan.claude-dev" {
				t.Fatalf("ext=%s", e.ExtensionID)
			}
			if e.Active != nil {
				t.Fatal("active must remain UNKNOWN (nil) without runtime evidence")
			}
			if _, ok := artifactFromEntry(e).Payload["ai_active"]; ok {
				t.Fatal("ai_active forbidden")
			}
		}
	}
	if !found {
		t.Fatalf("cline not found in %+v", got)
	}
}

func TestEngineIgnoresNonAIExtension(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "ms-python.python-2024.0.0")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{"name":"python","publisher":"ms-python","version":"2024.0.0","displayName":"Python"}`
	if err := os.WriteFile(filepath.Join(extDir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: "/Applications/Cursor.app", Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
			SignatureChecked: true, SignatureValid: true,
		}}},
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{
				RootsByProduct: map[string][]string{identity.ProductCursorID: {root}},
			},
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got {
		if e.Identification.Product != nil && e.Identification.Product.ID == "ms-python" {
			t.Fatal("python must not appear")
		}
		if e.ExtensionID == "ms-python.python" {
			t.Fatal("non-AI extension must not be inventoried")
		}
	}
}
