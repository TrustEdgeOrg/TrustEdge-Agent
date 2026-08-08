package identity

// builtinProducts is the versionable known-AI catalog.
//
// Values marked "verified" were confirmed from an authentic local Cursor.app
// installation via Info.plist and Security-framework code-signing APIs
// (not via the codesign CLI as the production extractor). Do not invent
// bundle IDs, Team IDs, or signing identifiers.
func builtinProducts() []KnownAIProduct {
	return []KnownAIProduct{
		cursorProduct(),
	}
}

// cursorProduct is catalog ID "cursor".
//
// Verified 2026-08-08 from /Applications/Cursor.app on a developer machine:
//   - CFBundleIdentifier / signing identifier: com.todesktop.230313mzl4w4u92
//   - TeamIdentifier: VDXQ22DGB9
//   - CFBundleExecutable: Cursor
//   - Leaf cert subject includes: Developer ID Application: Hilary Stout (VDXQ22DGB9)
//
// CandidatePaths are discovery hints only (not strong identity).
// ExpectedHashes left empty: binary digests change per release and are unresolved.
func cursorProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:       "cursor",
		Name:     "Cursor",
		Vendor:   "Cursor",
		Category: ProductCategoryCodeEditor,

		CandidateNames: []string{
			"Cursor",
			"Cursor Helper",
			"Cursor Helper (GPU)",
			"Cursor Helper (Renderer)",
			"Cursor Helper (Plugin)",
		},
		CandidatePaths: []string{
			"/Applications/Cursor.app",
			"~/Applications/Cursor.app",
		},

		BundleIDs: []string{
			"com.todesktop.230313mzl4w4u92",
		},
		SigningIdentifiers: []string{
			"com.todesktop.230313mzl4w4u92",
		},
		TeamIDs: []string{
			"VDXQ22DGB9",
		},
		// ExpectedHashes: unresolved — not pinned to a single release binary.
		ExpectedHashes: nil,
	}
}

// ProductCursorID is the stable catalog key for Cursor.
const ProductCursorID = "cursor"
