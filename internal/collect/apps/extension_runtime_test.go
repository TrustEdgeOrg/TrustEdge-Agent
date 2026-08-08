package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestExtensionActiveUnknownWithSharedHost(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "Continue.continue-1.0.0")
	_ = os.MkdirAll(extDir, 0o755)
	_ = os.WriteFile(filepath.Join(extDir, "package.json"), []byte(`{"name":"continue","publisher":"Continue","version":"1.0.0"}`), 0o644)

	cursorPath := "/Applications/Cursor.app"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: cursorPath, Executable: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92", TeamID: "VDXQ22DGB9",
			SignatureChecked: true, SignatureValid: true,
		}}},
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 1, Executable: cursorPath + "/Contents/MacOS/Cursor", Comm: "Cursor"},
				{PID: 2, PPID: 1, Executable: cursorPath + "/Contents/Frameworks/Cursor Helper (Plugin).app/Contents/MacOS/Cursor Helper (Plugin)", Comm: "Cursor Helper (Plugin): extension-host"},
				{PID: 3, PPID: 2, Executable: "/bin/zsh", Comm: "zsh"},
				{PID: 4, PPID: 3, Executable: "/usr/bin/git", Comm: "git"},
			}, nil
		},
		ExtensionProviders: []ExtensionProvider{
			&VSCodeCompatibleExtensionProvider{RootsByProduct: map[string][]string{identity.ProductCursorID: {root}}},
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	var cont *InventoryEntry
	for i := range got {
		if got[i].Identification.Product != nil && got[i].Identification.Product.ID == "continue" {
			cont = &got[i]
			break
		}
	}
	if cont == nil {
		t.Fatal("continue missing")
	}
	if cont.Active != nil {
		t.Fatalf("shared extension host must leave active UNKNOWN, got %v", *cont.Active)
	}
	if cont.Running {
		t.Fatal("extension row must not be Running just because host is")
	}
}
