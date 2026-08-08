package identity

// builtinExtensionProducts returns verified VS Code-compatible AI extensions.
//
// ExtensionIDs are verified from upstream package.json / marketplace itemName.
// Do not invent publishers or package names. Cursor built-ins (anysphere.cursor-*)
// are intentionally excluded — they are part of the Cursor app product.
func builtinExtensionProducts() []KnownAIProduct {
	return []KnownAIProduct{
		githubCopilotExtension(),
		continueExtension(),
		clineExtension(),
		rooCodeExtension(),
	}
}

func githubCopilotExtension() KnownAIProduct {
	// Marketplace itemName: GitHub.copilot
	// https://marketplace.visualstudio.com/items?itemName=GitHub.copilot
	return KnownAIProduct{
		ID:       "github_copilot",
		Name:     "GitHub Copilot",
		Vendor:   "GitHub",
		Category: ProductCategoryAIIDEExtension,
		ExtensionIDs: []string{
			"GitHub.copilot",
		},
		HostIDEProductIDs: []string{ProductCursorID, ProductVSCodeID},
		CandidateNames:    []string{"copilot"},
	}
}

func continueExtension() KnownAIProduct {
	// Verified from continuedev/continue extensions/vscode/package.json:
	// publisher=Continue name=continue → Continue.continue
	return KnownAIProduct{
		ID:       "continue",
		Name:     "Continue",
		Vendor:   "Continue",
		Category: ProductCategoryAIIDEExtension,
		ExtensionIDs: []string{
			"Continue.continue",
		},
		HostIDEProductIDs: []string{ProductCursorID, ProductVSCodeID},
		CandidateNames:    []string{"continue"},
	}
}

func clineExtension() KnownAIProduct {
	// Verified from cline/cline apps/vscode/package.json:
	// publisher=saoudrizwan name=claude-dev displayName=Cline
	// Agentic: multi-step coding agent with tools/MCP per product docs.
	return KnownAIProduct{
		ID:       "cline",
		Name:     "Cline",
		Vendor:   "Cline",
		Category: ProductCategoryAgenticIDEExtension,
		ExtensionIDs: []string{
			"saoudrizwan.claude-dev",
		},
		HostIDEProductIDs: []string{ProductCursorID, ProductVSCodeID},
		CandidateNames:    []string{"claude-dev", "Cline"},
	}
}

func rooCodeExtension() KnownAIProduct {
	// Verified from RooCodeInc/Roo-Code src/package.json:
	// publisher=RooVeterinaryInc name=roo-cline
	return KnownAIProduct{
		ID:       "roo_code",
		Name:     "Roo Code",
		Vendor:   "Roo Code",
		Category: ProductCategoryAgenticIDEExtension,
		ExtensionIDs: []string{
			"RooVeterinaryInc.roo-cline",
		},
		HostIDEProductIDs: []string{ProductCursorID, ProductVSCodeID},
		CandidateNames:    []string{"roo-cline", "Roo Code"},
	}
}
