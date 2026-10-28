package apps

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const maxSkillsPerRoot = 200

// AgentSkill is metadata for an installed skill definition. Skill contents are
// deliberately never included in telemetry.
type AgentSkill struct {
	ID     string
	Name   string
	Source string
	Scope  string
}

type skillRoot struct {
	path   string
	source string
}

func (e *Engine) attachAgentSkills(byPath map[string]*InventoryEntry) {
	home, err := homeDirFn()
	if err != nil || strings.TrimSpace(home) == "" {
		return
	}

	seen := make(map[*InventoryEntry]struct{})
	for _, entry := range byPath {
		if entry == nil || entry.Identification.Product == nil {
			continue
		}
		category := entry.Identification.Product.Category
		if category != identity.ProductCategoryCodeEditor && category != identity.ProductCategoryCLIAgent {
			continue
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		entry.Skills = discoverSkillsForProduct(home, entry.Identification.Product.ID)
	}
}

func discoverSkillsForProduct(home, productID string) []AgentSkill {
	var skills []AgentSkill
	seen := make(map[string]struct{})
	for _, root := range skillRoots(home, productID) {
		entries, err := os.ReadDir(root.path)
		if err != nil {
			continue
		}
		for index, entry := range entries {
			if index >= maxSkillsPerRoot {
				break
			}
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			folder := strings.TrimSpace(entry.Name())
			if folder == "" {
				continue
			}
			key := strings.ToLower(root.source + ":" + folder)
			if _, ok := seen[key]; ok {
				continue
			}
			definition := filepath.Join(root.path, folder, "SKILL.md")
			if info, err := os.Stat(definition); err != nil || !info.Mode().IsRegular() {
				continue
			}
			name := skillDefinitionName(definition, folder)
			skills = append(skills, AgentSkill{
				ID:     root.source + ":" + folder,
				Name:   name,
				Source: root.source,
				Scope:  "user",
			})
			seen[key] = struct{}{}
		}
	}
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Source != skills[j].Source {
			return skills[i].Source < skills[j].Source
		}
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})
	return skills
}

func skillRoots(home, productID string) []skillRoot {
	roots := []skillRoot{
		{path: filepath.Join(home, ".agents", "skills"), source: "agents"},
	}
	switch strings.ToLower(productID) {
	case identity.ProductCursorID:
		roots = append(roots,
			skillRoot{path: filepath.Join(home, ".claude", "skills"), source: "claude"},
			skillRoot{path: filepath.Join(home, ".cursor", "skills"), source: "cursor"},
			skillRoot{path: cursorUserDataDir(home), source: "cursor"},
		)
	case identity.ProductVSCodeID:
		roots = append(roots,
			skillRoot{path: filepath.Join(home, ".claude", "skills"), source: "claude"},
			skillRoot{path: filepath.Join(home, ".copilot", "skills"), source: "copilot"},
			skillRoot{path: vscodeUserDataDir(home), source: "vscode"},
		)
	case identity.ProductClaudeCodeID:
		roots = append(roots, skillRoot{path: filepath.Join(home, ".claude", "skills"), source: "claude"})
	case identity.ProductCodexCLIID:
		roots = append(roots, skillRoot{path: filepath.Join(home, ".codex", "skills"), source: "codex"})
	case identity.ProductGeminiCLIID:
		roots = append(roots, skillRoot{path: filepath.Join(home, ".gemini", "skills"), source: "gemini"})
	case identity.ProductCopilotCLIID:
		roots = append(roots, skillRoot{path: filepath.Join(home, ".copilot", "skills"), source: "copilot"})
	}
	return roots
}

func cursorUserDataDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "skills")
	case "windows":
		return filepath.Join(roamingAppDataDir(home), "Cursor", "User", "skills")
	default:
		config := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if config == "" {
			config = filepath.Join(home, ".config")
		}
		return filepath.Join(config, "Cursor", "User", "skills")
	}
}

func vscodeUserDataDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User", "skills")
	case "windows":
		return filepath.Join(roamingAppDataDir(home), "Code", "User", "skills")
	default:
		config := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if config == "" {
			config = filepath.Join(home, ".config")
		}
		return filepath.Join(config, "Code", "User", "skills")
	}
}

func roamingAppDataDir(home string) string {
	if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
		return appData
	}
	return filepath.Join(home, "AppData", "Roaming")
}

func skillDefinitionName(path, fallback string) string {
	file, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 8192)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return fallback
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "name" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")
		if value != "" {
			if len(value) > 128 {
				value = value[:128]
			}
			return value
		}
	}
	return fallback
}
