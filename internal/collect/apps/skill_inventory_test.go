package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestDiscoverSkillsForIDEReadsNamesOnlyFromKnownRoots(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, filepath.Join(home, ".agents", "skills", "release-check"), "release-check", "Release Check")
	writeSkill(t, filepath.Join(home, ".cursor", "skills", "review"), "review", "Review")
	if err := os.MkdirAll(filepath.Join(home, ".cursor", "skills", "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "projects", "repo", ".cursor", "skills", "workspace-only"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := discoverSkillsForProduct(home, identity.ProductCursorID)
	if len(got) != 2 {
		t.Fatalf("skills=%+v", got)
	}
	if got[0].ID != "agents:release-check" || got[0].Name != "Release Check" {
		t.Fatalf("first skill=%+v", got[0])
	}
	if got[1].ID != "cursor:review" || got[1].Name != "Review" {
		t.Fatalf("second skill=%+v", got[1])
	}
	if got[0].Scope != "user" || got[0].Source != "agents" {
		t.Fatalf("skill metadata=%+v", got[0])
	}
}

func TestDiscoverSkillsForIDEUsesEditorSpecificRoots(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, filepath.Join(home, ".copilot", "skills", "explain"), "explain", "Explain")
	writeSkill(t, filepath.Join(home, ".cursor", "skills", "pair"), "pair", "Pair")

	got := discoverSkillsForProduct(home, identity.ProductVSCodeID)
	if len(got) != 1 || got[0].ID != "copilot:explain" {
		t.Fatalf("VS Code skills=%+v", got)
	}
}

func TestDiscoverSkillsForCLIAgent(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, filepath.Join(home, ".agents", "skills", "shared"), "shared", "Shared")
	writeSkill(t, filepath.Join(home, ".codex", "skills", "refactor"), "refactor", "Refactor")
	writeSkill(t, filepath.Join(home, ".claude", "skills", "claude-only"), "claude-only", "Claude only")

	got := discoverSkillsForProduct(home, identity.ProductCodexCLIID)
	if len(got) != 2 || got[0].ID != "agents:shared" || got[1].ID != "codex:refactor" {
		t.Fatalf("Codex skills=%+v", got)
	}
}

func TestSkillDefinitionNameFallsBackToDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(path, []byte("# no frontmatter\nprivate body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := skillDefinitionName(path, "folder-name"); got != "folder-name" {
		t.Fatalf("name=%q", got)
	}
}

func TestArtifactIncludesSkillMetadata(t *testing.T) {
	entry := InventoryEntry{
		Identity: identity.ApplicationIdentity{Path: "/Applications/Cursor.app"},
		Identification: identity.IdentificationResult{
			Product: &identity.KnownAIProduct{
				ID:       identity.ProductCursorID,
				Name:     "Cursor",
				Vendor:   "Cursor",
				Category: identity.ProductCategoryCodeEditor,
			},
		},
		Installed: true,
		Skills:    []AgentSkill{{ID: "agents:review", Name: "Review", Source: "agents", Scope: "user"}},
	}
	art := artifactFromEntry(entry)
	skills, ok := art.Payload["skills"].([]any)
	if !ok || len(skills) != 1 {
		t.Fatalf("skills=%v", art.Payload["skills"])
	}
	skill, ok := skills[0].(map[string]any)
	if !ok {
		t.Fatalf("skill=%T", skills[0])
	}
	if skill["name"] != "Review" || skill["source"] != "agents" || skill["scope"] != "user" {
		t.Fatalf("skill=%v", skill)
	}
	if _, hasContents := skill["contents"]; hasContents {
		t.Fatalf("skill contents leaked: %v", skill)
	}
}

func writeSkill(t *testing.T, dir, folder, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	frontmatter := "---\nname: " + name + "\ndescription: do not send this\n---\nprivate instructions\n"
	if name == "" {
		frontmatter = "private instructions\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(frontmatter), 0o600); err != nil {
		t.Fatal(err)
	}
}
