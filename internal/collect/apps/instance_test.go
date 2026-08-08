package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestGroupInstancesElectronHelpers(t *testing.T) {
	product := cursorProductPtr()
	bundle := "/Applications/Cursor.app"
	entry := InventoryEntry{
		Identity: identity.ApplicationIdentity{
			Path:              bundle,
			BundleID:          "com.todesktop.230313mzl4w4u92",
			Executable:        "Cursor",
			ExecutablePath:    bundle + "/Contents/MacOS/Cursor",
			SigningIdentifier: "com.todesktop.230313mzl4w4u92",
			TeamID:            "VDXQ22DGB9",
			SignatureValid:    true,
			SignatureChecked:  true,
		},
		Identification: identity.IdentificationResult{
			Product:    product,
			Confidence: identity.ConfidenceVerified,
		},
		Installed: true,
		Running:   true,
	}
	procs := []process.ProcessInfo{
		{PID: 1, PPID: 0, Comm: "Cursor", Executable: bundle + "/Contents/MacOS/Cursor"},
		{PID: 2, PPID: 1, Comm: "Cursor Helper", Executable: bundle + "/Contents/Frameworks/Cursor Helper.app/Contents/MacOS/Cursor Helper"},
		{PID: 3, PPID: 1, Comm: "Cursor Helper (Renderer)", Executable: bundle + "/Contents/Frameworks/Cursor Helper (Renderer).app/Contents/MacOS/Cursor Helper (Renderer)"},
		{PID: 4, PPID: 1, Comm: "Cursor Helper (GPU)", Executable: bundle + "/Contents/Frameworks/Cursor Helper (GPU).app/Contents/MacOS/Cursor Helper (GPU)"},
		{PID: 5, PPID: 1, Comm: "Cursor Helper (Plugin)", Executable: bundle + "/Contents/Frameworks/Cursor Helper (Plugin).app/Contents/MacOS/Cursor Helper (Plugin)"},
		// Arbitrary descendant outside the bundle — must NOT be grouped.
		{PID: 6, PPID: 1, Comm: "node", Executable: "/usr/local/bin/node"},
	}

	insts := groupInstances([]InventoryEntry{entry}, procs)
	if len(insts) != 1 {
		t.Fatalf("len=%d", len(insts))
	}
	inst := insts[0]
	if len(inst.Processes) != 5 {
		t.Fatalf("processes=%d want 5 (node excluded): %+v", len(inst.Processes), inst.Processes)
	}
	roles := map[identity.ProcessRole]int{}
	for _, p := range inst.Processes {
		roles[p.Role]++
		if p.PID == 6 {
			t.Fatal("node must not be associated")
		}
	}
	if roles[identity.ProcessRoleMain] < 1 {
		t.Fatalf("roles=%v", roles)
	}
	if roles[identity.ProcessRoleRenderer] < 1 || roles[identity.ProcessRoleGPU] < 1 {
		t.Fatalf("expected renderer/gpu roles: %v", roles)
	}
}

func TestGroupInstancesRejectsAncestryAlone(t *testing.T) {
	product := cursorProductPtr()
	bundle := "/Applications/Cursor.app"
	entry := InventoryEntry{
		Identity: identity.ApplicationIdentity{Path: bundle, Executable: "Cursor"},
		Identification: identity.IdentificationResult{
			Product:    product,
			Confidence: identity.ConfidenceVerified,
		},
		Installed: true,
	}
	procs := []process.ProcessInfo{
		{PID: 10, PPID: 1, Comm: "Cursor", Executable: bundle + "/Contents/MacOS/Cursor"},
		{PID: 11, PPID: 10, Comm: "bash", Executable: "/bin/bash"},
	}
	inst := groupInstances([]InventoryEntry{entry}, procs)[0]
	if len(inst.Processes) != 1 || inst.Processes[0].PID != 10 {
		t.Fatalf("got %+v", inst.Processes)
	}
}

func cursorProductPtr() *identity.KnownAIProduct {
	p, ok := identity.DefaultCatalog().Lookup(identity.ProductCursorID)
	if !ok {
		panic("cursor missing")
	}
	return &p
}
