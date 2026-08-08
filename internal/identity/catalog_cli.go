package identity

// CLI agent catalog entries.
//
// PackageManagers, PackageIdentifiers, SigningIdentifiers, TeamIDs, and
// ExpectedHashes are intentionally empty (unresolved). Do not invent npm
// package names, Homebrew formulas, or Apple Team IDs. Until those are
// verified from an authentic install or official metadata checked into this
// repo, CLI matches must not reach VERIFIED on command name alone.
//
// CandidateNames / ExecutableNames are discovery hints only.

func builtinCLIProducts() []KnownAIProduct {
	return []KnownAIProduct{
		claudeCodeCLIProduct(),
		codexCLIProduct(),
		geminiCLIProduct(),
		copilotCLIProduct(),
		opencodeCLIProduct(),
	}
}

func claudeCodeCLIProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductClaudeCodeID,
		Name:            "Claude Code",
		Vendor:          "Anthropic",
		Category:        ProductCategoryCLIAgent,
		ExecutableNames: []string{"claude"},
		// CandidateNames left empty: EqualFold("Claude","claude") would collide
		// with Claude Desktop. Exact ExecutableNames only.
		// Package/signing evidence unresolved — not invented.
	}
}

func codexCLIProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductCodexCLIID,
		Name:            "OpenAI Codex CLI",
		Vendor:          "OpenAI",
		Category:        ProductCategoryCLIAgent,
		ExecutableNames: []string{"codex"},
	}
}

func geminiCLIProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductGeminiCLIID,
		Name:            "Gemini CLI",
		Vendor:          "Google",
		Category:        ProductCategoryCLIAgent,
		ExecutableNames: []string{"gemini"},
	}
}

func copilotCLIProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductCopilotCLIID,
		Name:            "GitHub Copilot CLI",
		Vendor:          "GitHub",
		Category:        ProductCategoryCLIAgent,
		ExecutableNames: []string{"copilot"},
	}
}

func opencodeCLIProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductOpenCodeID,
		Name:            "OpenCode",
		Vendor:          "OpenCode",
		Category:        ProductCategoryCLIAgent,
		ExecutableNames: []string{"opencode"},
	}
}

const (
	ProductClaudeCodeID = "claude_code"
	ProductCodexCLIID   = "codex_cli"
	ProductGeminiCLIID  = "gemini_cli"
	ProductCopilotCLIID = "copilot_cli"
	ProductOpenCodeID   = "opencode"
)
