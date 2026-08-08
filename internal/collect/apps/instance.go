package apps

import (
	"sort"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// Instances builds ApplicationInstance groupings from the current inventory
// and process table. Only processes whose executables live under a verified
// (or strongly identified) application bundle are associated. Ancestry alone
// never pulls in arbitrary descendants such as node.
func (e *Engine) Instances() ([]identity.ApplicationInstance, error) {
	entries, err := e.Inventory()
	if err != nil {
		return nil, err
	}
	procs, err := e.listProcs()
	if err != nil {
		e.logf("known-ai instances process list: %v", err)
		procs = nil
	}
	return groupInstances(entries, procs), nil
}

func groupInstances(entries []InventoryEntry, procs []process.ProcessInfo) []identity.ApplicationInstance {
	var out []identity.ApplicationInstance
	for _, entry := range entries {
		if entry.Identification.Product == nil {
			continue
		}
		// Require stronger than name-only before grouping helpers.
		if identityRank(entry.Identification.Confidence) < identityRank(identity.ConfidenceMedium) &&
			!entry.Installed {
			// Name-only running binary: single-process instance, no helper association.
			inst := baseInstance(entry)
			for _, proc := range procs {
				if proc.PID > 0 && sameExecutable(proc.Executable, entry.Identity.ExecutablePath) {
					inst.Processes = append(inst.Processes, identity.InstanceProcess{
						PID:        proc.PID,
						PPID:       proc.PPID,
						Executable: proc.Executable,
						Comm:       proc.Comm,
						Role:       classifyRole(proc.Comm, proc.Executable),
					})
				}
			}
			inst.Running = len(inst.Processes) > 0
			out = append(out, inst)
			continue
		}

		bundle := entry.Identity.Path
		if !strings.HasSuffix(strings.ToLower(bundle), ".app") {
			bundle = EnclosingAppPath(entry.Identity.ExecutablePath)
		}
		inst := baseInstance(entry)
		inst.BundlePath = bundle

		byPID := make(map[int]process.ProcessInfo, len(procs))
		for _, proc := range procs {
			byPID[proc.PID] = proc
		}

		for _, proc := range procs {
			if !belongsToBundle(proc, bundle, entry, byPID) {
				continue
			}
			inst.Processes = append(inst.Processes, identity.InstanceProcess{
				PID:        proc.PID,
				PPID:       proc.PPID,
				Executable: proc.Executable,
				Comm:       proc.Comm,
				Role:       classifyRole(proc.Comm, proc.Executable),
			})
		}
		sort.Slice(inst.Processes, func(i, j int) bool {
			return inst.Processes[i].PID < inst.Processes[j].PID
		})
		inst.Running = len(inst.Processes) > 0
		out = append(out, inst)
	}
	return out
}

func baseInstance(entry InventoryEntry) identity.ApplicationInstance {
	return identity.ApplicationInstance{
		Product:    entry.Identification.Product,
		Identity:   entry.Identity,
		Confidence: entry.Identification.Confidence,
		BundlePath: entry.Identity.Path,
		Installed:  entry.Installed,
		Running:    entry.Running,
		Matched:    append([]identity.EvidenceKey(nil), entry.Identification.Matched...),
		Failed:     append([]identity.EvidenceKey(nil), entry.Identification.Failed...),
	}
}

// belongsToBundle requires executable provenance under the .app bundle.
// Parent ancestry may support association only when the child executable is
// also inside the same bundle (Electron helpers), never for external tools.
func belongsToBundle(proc process.ProcessInfo, bundle string, entry InventoryEntry, byPID map[int]process.ProcessInfo) bool {
	exe := strings.TrimSpace(proc.Executable)
	if exe == "" || bundle == "" {
		return false
	}
	if executableWithinBundle(exe, bundle) {
		return true
	}
	return false
}

func executableWithinBundle(exe, bundle string) bool {
	exe = posixPath(exe)
	bundle = posixPath(bundle)
	if exe == "" || bundle == "" || exe == "." || bundle == "." {
		return false
	}
	if exe == bundle {
		return true
	}
	prefix := bundle + "/"
	return strings.HasPrefix(exe, prefix)
}

func classifyRole(comm, exe string) identity.ProcessRole {
	name := strings.ToLower(strings.TrimSpace(comm))
	if name == "" {
		name = strings.ToLower(posixBase(exe))
	}
	switch {
	case strings.Contains(name, "extension"):
		return identity.ProcessRoleExtensionHost
	case strings.Contains(name, "renderer"):
		return identity.ProcessRoleRenderer
	case strings.Contains(name, "gpu"):
		return identity.ProcessRoleGPU
	case strings.Contains(name, "helper"):
		return identity.ProcessRoleHelper
	case name == "cursor" || !strings.Contains(name, "helper"):
		// Main app executable basename without Helper suffix.
		base := strings.ToLower(posixBase(exe))
		if strings.Contains(base, "helper") {
			return identity.ProcessRoleHelper
		}
		if strings.HasSuffix(base, " cursor") || base == "cursor" || !strings.Contains(base, " ") {
			// Prefer main when basename equals product executable.
			if !strings.Contains(base, "helper") && !strings.Contains(base, "renderer") {
				return identity.ProcessRoleMain
			}
		}
		return identity.ProcessRoleMain
	default:
		return identity.ProcessRoleOther
	}
}

func sameExecutable(a, b string) bool {
	a, b = posixPath(a), posixPath(b)
	return a != "." && b != "." && a != "" && b != "" && strings.EqualFold(a, b)
}

func identityRank(c identity.Confidence) int {
	switch c {
	case identity.ConfidenceVerified:
		return 5
	case identity.ConfidenceHigh:
		return 4
	case identity.ConfidenceMedium:
		return 3
	case identity.ConfidenceLow:
		return 2
	default:
		return 1
	}
}
