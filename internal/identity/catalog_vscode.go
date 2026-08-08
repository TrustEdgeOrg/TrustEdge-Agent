package identity

// Visual Studio Code host IDE catalog entry.
//
// Bundle ID is the stable CFBundleIdentifier published for the official macOS
// VS Code app (com.microsoft.VSCode). TeamID / signing pins left empty until
// verified from an authentic local install on this project.
func vscodeProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:       ProductVSCodeID,
		Name:     "Visual Studio Code",
		Vendor:   "Microsoft",
		Category: ProductCategoryCodeEditor,

		CandidateNames: []string{
			"Code",
			"Visual Studio Code",
			"Code Helper",
			"Code Helper (GPU)",
			"Code Helper (Renderer)",
			"Code Helper (Plugin)",
		},
		CandidatePaths: []string{
			"/Applications/Visual Studio Code.app",
			"~/Applications/Visual Studio Code.app",
		},
		BundleIDs: []string{
			"com.microsoft.VSCode",
		},
		// SigningIdentifiers / TeamIDs unresolved — not invented.
	}
}
