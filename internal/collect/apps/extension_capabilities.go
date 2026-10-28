package apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

// mcpServersForHost returns configured MCP server names from device-scoped mcp.json
// files (mcpServers keys only — env/secrets are never read).
func mcpServersForHost(home, hostProductID string) []string {
	seen := make(map[string]struct{})
	for _, p := range mcpConfigCandidates(home, hostProductID) {
		for _, name := range mcpServerNamesFromFile(p) {
			seen[name] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func mcpServerNamesFromFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.MCPServers) == 0 {
		return nil
	}
	out := make([]string, 0, len(doc.MCPServers))
	for name := range doc.MCPServers {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func attachMCPInventory(eptr *InventoryEntry, home, hostProductID string, discovered map[string][]string) {
	if eptr == nil {
		return
	}
	if len(discovered) > 0 {
		eptr.MCPServerTools = discovered
	}
	if servers := mcpServersForHost(home, hostProductID); len(servers) > 0 {
		eptr.MCPConfigured = true
		eptr.MCPServers = servers
		if !hasEvidence(eptr.Identification.Matched, identity.EvidenceMCPConfigured) {
			eptr.Identification.Matched = append(eptr.Identification.Matched, identity.EvidenceMCPConfigured)
		}
		return
	}
	if mcpConfiguredForHost(home, hostProductID) {
		eptr.MCPConfigured = true
		if !hasEvidence(eptr.Identification.Matched, identity.EvidenceMCPConfigured) {
			eptr.Identification.Matched = append(eptr.Identification.Matched, identity.EvidenceMCPConfigured)
		}
	}

}

func (e *Engine) attachExtensionCapabilities(byPath map[string]*InventoryEntry) {
	home, _ := homeDirFn()
	discovered, cacheKey := discoverAllMCPToolsAtHome(home)
	if cacheKey != "" && cacheKey == e.mcpToolsKey && len(e.mcpToolsCache) > 0 {
		discovered = e.mcpToolsCache
	} else if len(discovered) > 0 {
		e.mcpToolsKey = cacheKey
		e.mcpToolsCache = discovered
	}
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
		attachMCPInventory(eptr, home, eptr.HostIDEProductID, discovered)
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

	// Host IDEs (Cursor, VS Code) — MCP config is device-scoped on the IDE, not per extension.
	for _, eptr := range byPath {
		if eptr == nil || eptr.Identification.Product == nil {
			continue
		}
		pid := eptr.Identification.Product.ID
		if pid != identity.ProductCursorID && pid != identity.ProductVSCodeID {
			continue
		}
		attachMCPInventory(eptr, home, pid, discovered)
	}
}
