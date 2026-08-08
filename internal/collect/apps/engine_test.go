package apps

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

type stubDiscoverer struct {
	apps []identity.ApplicationIdentity
}

func (s stubDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	return s.apps, nil
}

type stubSigner struct {
	info SigningInfo
}

func (s stubSigner) Extract(path string) (SigningInfo, error) {
	_ = path
	return s.info, nil
}

func (s stubSigner) Validate(path string) (bool, error) {
	_ = path
	return s.info.SignatureValid, nil
}

func TestEngineInstalledAndRunning(t *testing.T) {
	appPath := "/Applications/Cursor.app"
	disc := stubDiscoverer{apps: []identity.ApplicationIdentity{{
		Path:           appPath,
		BundleID:       "com.todesktop.230313mzl4w4u92",
		Executable:     "Cursor",
		ExecutablePath: appPath + "/Contents/MacOS/Cursor",
		Version:        "1.0",
	}}}
	signer := stubSigner{info: SigningInfo{
		SigningIdentifier:  "com.todesktop.230313mzl4w4u92",
		TeamID:             "VDXQ22DGB9",
		CertificateSubject: "Developer ID Application: Hilary Stout (VDXQ22DGB9)",
		SignatureValid:     true,
		SignatureChecked:   true,
	}}
	// ExtractAndValidate will call Validate separately; stub returns valid.
	signer.info.SignatureChecked = false // Extract alone
	// Use a custom signer that sets flags via ExtractAndValidate path:
	full := stubSigner{info: SigningInfo{
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
	}}

	eng := NewEngine(EngineConfig{
		Logger:     log.Default(),
		Discoverer: disc,
		Signer:     full,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{
				PID:        42,
				Executable: appPath + "/Contents/MacOS/Cursor",
				Comm:       "Cursor",
			}}, nil
		},
	})

	inv, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(inv) != 1 {
		t.Fatalf("len=%d", len(inv))
	}
	e := inv[0]
	if !e.Installed || !e.Running {
		t.Fatalf("installed=%v running=%v", e.Installed, e.Running)
	}
	if e.Identification.Confidence != identity.ConfidenceVerified {
		t.Fatalf("Confidence=%s failed=%v", e.Identification.Confidence, e.Identification.Failed)
	}
	if len(e.PIDs) != 1 || e.PIDs[0] != 42 {
		t.Fatalf("PIDs=%v", e.PIDs)
	}
}

func TestEngineInstalledNotRunning(t *testing.T) {
	appPath := "/Applications/Cursor.app"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path:       appPath,
			BundleID:   "com.todesktop.230313mzl4w4u92",
			Executable: "Cursor",
		}}},
		Signer: stubSigner{info: SigningInfo{
			SigningIdentifier: "com.todesktop.230313mzl4w4u92",
			TeamID:            "VDXQ22DGB9",
			SignatureValid:    true,
		}},
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	inv, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(inv) != 1 {
		t.Fatalf("len=%d", len(inv))
	}
	if !inv[0].Installed || inv[0].Running {
		t.Fatalf("installed=%v running=%v", inv[0].Installed, inv[0].Running)
	}
}

func TestEngineNameOnlyExecutableNotVerified(t *testing.T) {
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     stubSigner{},
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{
				PID:        7,
				Executable: "/tmp/Cursor",
				Comm:       "Cursor",
			}}, nil
		},
	})
	inv, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(inv) != 1 {
		t.Fatalf("len=%d want name-only candidate entry", len(inv))
	}
	if inv[0].Installed {
		t.Fatal("bare executable must not set installed")
	}
	if inv[0].Identification.Confidence == identity.ConfidenceVerified {
		t.Fatal("name-only must not be VERIFIED")
	}
}

func TestMonitorBaselineThenDelta(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "Cursor.app")
	writeTestBundle(t, appPath)

	running := false
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path:       appPath,
			BundleID:   "com.todesktop.230313mzl4w4u92",
			Executable: "Cursor",
		}}},
		Signer: stubSigner{info: SigningInfo{
			SigningIdentifier: "com.todesktop.230313mzl4w4u92",
			TeamID:            "VDXQ22DGB9",
			SignatureValid:    true,
		}},
		ListProcs: func() ([]process.ProcessInfo, error) {
			if !running {
				return nil, nil
			}
			return []process.ProcessInfo{{
				PID:        99,
				Executable: filepath.Join(appPath, "Contents", "MacOS", "Cursor"),
			}}, nil
		},
	})
	mon := NewMonitor(nil, eng)
	if changes := mon.Poll(); len(changes) != 0 {
		t.Fatalf("baseline should be silent, got %d", len(changes))
	}
	running = true
	changes := mon.Poll()
	if len(changes) != 1 {
		t.Fatalf("want running delta, got %d", len(changes))
	}
	if changes[0].Type != constants.TypeKnownAIApp {
		t.Fatalf("Type=%s", changes[0].Type)
	}
	if changes[0].Payload["running"] != true {
		t.Fatalf("payload=%v", changes[0].Payload)
	}
	if changes[0].Payload["installed"] != true {
		t.Fatal("expected installed true")
	}
}

func writeTestBundle(t *testing.T, appPath string) {
	t.Helper()
	contents := filepath.Join(appPath, "Contents")
	if err := os.MkdirAll(filepath.Join(contents, "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
}
