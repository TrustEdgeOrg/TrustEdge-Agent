package identity

// ApplicationIdentity is the extracted identity of a concrete application
// or executable on disk. Strong fields (bundle ID, signing ID, Team ID,
// signature validity, hash) establish product identity; names and paths
// are discovery hints only.
type ApplicationIdentity struct {
	// Path is the application bundle path (e.g. /Applications/Cursor.app)
	// or the executable path when no enclosing bundle is known.
	Path string

	// BundleID is CFBundleIdentifier from Info.plist when available.
	BundleID string

	// Version is CFBundleShortVersionString (or CFBundleVersion fallback).
	Version string

	// Executable is the main executable basename (CFBundleExecutable)
	// or the process executable basename.
	Executable string

	// ExecutablePath is the absolute path to the main executable when known.
	ExecutablePath string

	// SigningIdentifier is the code-signing identifier (e.g. from
	// SecCodeCopySigningInformation kSecCodeInfoIdentifier).
	SigningIdentifier string

	// TeamID is the Apple Developer Team ID from the signing identity.
	TeamID string

	// CertificateSubject is a human-readable leaf certificate subject
	// when available (e.g. "Developer ID Application: …").
	CertificateSubject string

	// SignatureValid is true only when static code validation succeeded.
	// False means invalid or not yet validated; see SignatureChecked.
	SignatureValid bool

	// SignatureChecked is true when a validation attempt completed
	// (success or failure). Distinguishes "not checked" from "invalid".
	SignatureChecked bool

	// SHA256 is the hex-encoded SHA-256 of the main executable when calculated.
	SHA256 string
}

// KnownAIProduct is a catalog entry for a known AI application.
// CandidateNames and CandidatePaths are used only for candidate generation.
// BundleIDs, SigningIdentifiers, TeamIDs, and ExpectedHashes are strong
// verification evidence.
type KnownAIProduct struct {
	// ID is a stable catalog key (e.g. "cursor").
	ID string

	Name     string
	Vendor   string
	Category ProductCategory

	// CandidateNames are executable or display names used only to select
	// match candidates (never sufficient for VERIFIED).
	CandidateNames []string

	// CandidatePaths are expected install locations used only for candidate
	// generation (never sufficient for VERIFIED).
	CandidatePaths []string

	// BundleIDs are accepted CFBundleIdentifier values.
	BundleIDs []string

	// SigningIdentifiers are accepted code-signing identifiers.
	SigningIdentifiers []string

	// TeamIDs are accepted Apple Developer Team IDs.
	TeamIDs []string

	// ExpectedHashes are optional hex SHA-256 digests of known good binaries.
	// Empty means hash matching is not required for this product version set.
	ExpectedHashes []string
}

// EvidenceKey names a single identity evidence factor.
type EvidenceKey string

const (
	EvidenceBundleID           EvidenceKey = "bundle_id"
	EvidenceSigningIdentifier  EvidenceKey = "signing_identifier"
	EvidenceTeamID             EvidenceKey = "team_id"
	EvidenceSignatureValid     EvidenceKey = "signature_valid"
	EvidenceSHA256             EvidenceKey = "sha256"
	EvidenceCandidateName      EvidenceKey = "candidate_name"
	EvidenceCandidatePath      EvidenceKey = "candidate_path"
)

// IdentificationResult is the outcome of matching an ApplicationIdentity
// against the known-AI catalog.
type IdentificationResult struct {
	// Product is set when a catalog entry was selected as the best match.
	// Nil when Confidence is UNKNOWN and no candidate was plausible.
	Product *KnownAIProduct

	Confidence Confidence

	// Matched lists evidence factors that agreed with the catalog entry.
	Matched []EvidenceKey

	// Failed lists evidence factors that were expected but missing or mismatched.
	Failed []EvidenceKey

	// Installed is true when the application was observed on disk.
	Installed bool

	// Running is true when at least one correlated process is alive.
	// Independent of Installed; both may be true.
	Running bool
}
