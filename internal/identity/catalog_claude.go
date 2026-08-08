package identity

// claudeProduct is catalog ID "claude" (Claude Desktop).
//
// Verified 2026-08-08 from Anthropic's official macOS zip
// (downloads.claude.ai/.../Claude-*.zip → Claude.app):
//   - CFBundleIdentifier / signing identifier: com.anthropic.claudefordesktop
//   - TeamIdentifier: Q6L2SF6YDW
//   - CFBundleExecutable: Claude
//   - Leaf cert subject: Developer ID Application: Anthropic PBC (Q6L2SF6YDW)
//   - Electron helpers under Contents/Frameworks: Claude Helper*.app
//
// CandidatePaths are discovery hints only (not strong identity).
// ExpectedHashes left empty: binary digests change per release and are unresolved.
//
// Homebrew cask also uninstalls helper id com.anthropic.claudefordesktop.helper;
// that helper id is not required for main-app VERIFIED matching (helpers are
// associated by living under the verified .app bundle).
func claudeProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:       "claude",
		Name:     "Claude",
		Vendor:   "Anthropic",
		Category: ProductCategoryChatClient,

		CandidateNames: []string{
			"Claude",
			"Claude Helper",
			"Claude Helper (GPU)",
			"Claude Helper (Renderer)",
			"Claude Helper (Plugin)",
		},
		CandidatePaths: []string{
			"/Applications/Claude.app",
			"~/Applications/Claude.app",
		},

		BundleIDs: []string{
			"com.anthropic.claudefordesktop",
		},
		SigningIdentifiers: []string{
			"com.anthropic.claudefordesktop",
		},
		TeamIDs: []string{
			"Q6L2SF6YDW",
		},
		// ExpectedHashes: unresolved — not pinned to a single release binary.
		ExpectedHashes: nil,
	}
}

// ProductClaudeID is the stable catalog key for Claude Desktop.
const ProductClaudeID = "claude"
