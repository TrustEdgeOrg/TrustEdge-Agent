package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// mcpConfigCandidates are device-scoped known paths only (no project recursion).
func mcpConfigCandidates(home, hostProductID string) []string {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil
	}
	switch strings.ToLower(hostProductID) {
	case identity.ProductCursorID:
		return []string{
			filepath.Join(home, ".cursor", "mcp.json"),
			filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "mcp.json"),
		}
	case identity.ProductVSCodeID:
		return []string{
			filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json"),
			filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "mcp.json"),
		}
	default:
		return nil
	}
}

func mcpConfiguredForHost(home, hostProductID string) bool {
	for _, p := range mcpConfigCandidates(home, hostProductID) {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() || fi.Size() == 0 {
			continue
		}
		return true
	}
	return false
}

func (e *Engine) attachExtensionCapabilities(byPath map[string]*InventoryEntry) {
	home, _ := homeDirFn()
	// Local model product ids currently serving.
	localModels := make(map[string]struct{})
	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		if eptr.Identification.Product.Category == identity.ProductCategoryLocalModelRuntime && eptr.Serving {
			localModels[eptr.Identification.Product.ID] = struct{}{}
		}
	}

	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		cat := eptr.Identification.Product.Category
		if cat != identity.ProductCategoryAIIDEExtension && cat != identity.ProductCategoryAgenticIDEExtension {
			continue
		}
		if mcpConfiguredForHost(home, eptr.HostIDEProductID) {
			eptr.MCPConfigured = true
			if !hasEvidence(eptr.Identification.Matched, identity.EvidenceMCPConfigured) {
				eptr.Identification.Matched = append(eptr.Identification.Matched, identity.EvidenceMCPConfigured)
			}
		}
		// Prefer Ollama when serving; otherwise first local model id.
		if _, ok := localModels[identity.ProductOllamaID]; ok {
			eptr.LocalModelProductID = identity.ProductOllamaID
		} else {
			for id := range localModels {
				eptr.LocalModelProductID = id
				break
			}
		}
	}
}
