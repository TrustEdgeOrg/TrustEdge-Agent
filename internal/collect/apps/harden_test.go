package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestHardenCacheInvalidationCrossPlatform(t *testing.T) {
	c := identity.NewCache(8)
	k1 := identity.FileFingerprint("/Applications/Cursor.app", 1, 10)
	k2 := identity.FileFingerprint("/Applications/Cursor.app", 2, 10)
	c.PutIdentity(k1, identity.ApplicationIdentity{Version: "1"})
	if _, ok := c.Get(k2); ok {
		t.Fatal("changed mtime must miss")
	}
	c.InvalidatePath("/Applications/Cursor.app")
	if _, ok := c.Get(k1); ok {
		t.Fatal("expected invalidate")
	}
}

func TestHardenDuplicateInstallationsCrossPlatform(t *testing.T) {
	a := "/Applications/Cursor.app"
	b := "/Users/x/Applications/Cursor.app"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{
			legitAppShared(a),
			legitAppShared(b),
		}},
		Signer: verifiedSignerShared(),
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{
				{PID: 1, Executable: a + "/Contents/MacOS/Cursor"},
				{PID: 2, Executable: b + "/Contents/MacOS/Cursor"},
			}, nil
		},
	})
	inv, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(inv) != 2 {
		t.Fatalf("want 2 installs, got %d", len(inv))
	}
}

func TestHardenPIDReuseCrossPlatform(t *testing.T) {
	bundle := "/Applications/Cursor.app"
	pid := 4242
	list := []process.ProcessInfo{{PID: pid, Executable: bundle + "/Contents/MacOS/Cursor"}}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{legitAppShared(bundle)}},
		Signer:     verifiedSignerShared(),
		ListProcs:  func() ([]process.ProcessInfo, error) { return list, nil },
	})
	inv1, _ := eng.Inventory()
	if len(inv1) != 1 || inv1[0].PIDs[0] != pid {
		t.Fatalf("first: %+v", inv1)
	}
	list = []process.ProcessInfo{{PID: pid, Executable: "/usr/bin/true", Comm: "true"}}
	inv2, _ := eng.Inventory()
	if inv2[0].Running {
		t.Fatal("PID reuse outside bundle must not keep running=true")
	}
}

func TestHardenMonitorRemovalCrossPlatform(t *testing.T) {
	bundle := "/Applications/Cursor.app"
	present := true
	eng := NewEngine(EngineConfig{
		Discoverer: discovererFunc(func() ([]identity.ApplicationIdentity, error) {
			if !present {
				return nil, nil
			}
			return []identity.ApplicationIdentity{legitAppShared(bundle)}, nil
		}),
		Signer:    verifiedSignerShared(),
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	mon := NewMonitor(nil, eng)
	_ = mon.Poll()
	present = false
	changes := mon.Poll()
	if len(changes) != 1 || changes[0].Payload["removed"] != true {
		t.Fatalf("changes=%v", changes)
	}
}

func legitAppShared(path string) identity.ApplicationIdentity {
	return identity.ApplicationIdentity{
		Path:           path,
		BundleID:       "com.todesktop.230313mzl4w4u92",
		Executable:     "Cursor",
		ExecutablePath: path + "/Contents/MacOS/Cursor",
		Version:        "1.0",
	}
}

func verifiedSignerShared() fakeSigner {
	return fakeSigner{info: SigningInfo{
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
	}}
}
