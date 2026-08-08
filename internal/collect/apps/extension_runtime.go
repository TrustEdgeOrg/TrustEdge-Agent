package apps

import (
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// ExtensionToolAttribution is a conservatively attributed child process under
// an IDE extension host. Attribution is never VERIFIED from shared hosts alone.
type ExtensionToolAttribution struct {
	PID                  int
	Executable           string
	Comm                 string
	AttributionConfidence identity.Confidence
	CandidateExtensionID string
}

// correlateExtensionRuntime links extension rows to host IDE runtime state.
// Shared extension-host processes must NOT set Active=true for a specific extension.
func (e *Engine) correlateExtensionRuntime(byPath map[string]*InventoryEntry, procs []process.ProcessInfo) {
	hostRunning := make(map[string]bool)       // product id → running
	hostExtHost := make(map[string]bool)       // product id → extension host seen
	hostChildren := make(map[string][]ExtensionToolAttribution)

	byPID := make(map[int]process.ProcessInfo, len(procs))
	for _, p := range procs {
		byPID[p.PID] = p
	}

	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		p := eptr.Identification.Product
		if p.Category != identity.ProductCategoryCodeEditor {
			continue
		}
		if p.ID != identity.ProductCursorID && p.ID != identity.ProductVSCodeID {
			continue
		}
		if eptr.Running {
			hostRunning[p.ID] = true
		}
	}

	for _, proc := range procs {
		role := classifyRole(proc.Comm, proc.Executable)
		hostID := hostIDEProductForProcess(byPath, proc)
		if hostID == "" {
			continue
		}
		if role == identity.ProcessRoleExtensionHost {
			hostExtHost[hostID] = true
			continue
		}
		// Child of extension host: ambiguous which extension caused it.
		parent, ok := byPID[proc.PPID]
		if !ok {
			continue
		}
		if classifyRole(parent.Comm, parent.Executable) != identity.ProcessRoleExtensionHost {
			continue
		}
		hostChildren[hostID] = append(hostChildren[hostID], ExtensionToolAttribution{
			PID:                   proc.PID,
			Executable:            proc.Executable,
			Comm:                  proc.Comm,
			AttributionConfidence: identity.ConfidenceMedium,
		})
	}

	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		cat := eptr.Identification.Product.Category
		if cat != identity.ProductCategoryAIIDEExtension && cat != identity.ProductCategoryAgenticIDEExtension {
			continue
		}
		hostID := eptr.HostIDEProductID
		// Extensions are not themselves OS processes.
		eptr.Running = false
		// Active remains UNKNOWN: shared extension host is insufficient.
		eptr.Active = nil
		_ = hostRunning[hostID]
		_ = hostExtHost[hostID]
		_ = hostChildren[hostID]
		// Evidence note: we observed host/extension-host but do not claim activation.
		if hostExtHost[hostID] && !hasEvidence(eptr.Identification.Failed, identity.EvidenceExtensionActive) {
			eptr.Identification.Failed = append(eptr.Identification.Failed, identity.EvidenceExtensionActive)
		}
	}
}

func hostIDEProductForProcess(byPath map[string]*InventoryEntry, proc process.ProcessInfo) string {
	exe := posixPath(proc.Executable)
	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		p := eptr.Identification.Product
		if p.Category != identity.ProductCategoryCodeEditor {
			continue
		}
		if p.ID != identity.ProductCursorID && p.ID != identity.ProductVSCodeID {
			continue
		}
		bundle := eptr.Identity.Path
		if bundle != "" && executableWithinBundle(exe, bundle) {
			return p.ID
		}
		// Cursor/Code helpers often live under the .app Contents path.
		if bundle != "" && strings.HasPrefix(exe, strings.TrimSuffix(bundle, "/")+"/") {
			return p.ID
		}
		base := strings.ToLower(filepath.Base(exe))
		comm := strings.ToLower(proc.Comm)
		switch p.ID {
		case identity.ProductCursorID:
			if strings.Contains(base, "cursor") || strings.Contains(comm, "cursor") {
				return p.ID
			}
		case identity.ProductVSCodeID:
			if base == "code" || strings.HasPrefix(comm, "code") || strings.Contains(base, "visual studio code") {
				return p.ID
			}
		}
	}
	return ""
}
