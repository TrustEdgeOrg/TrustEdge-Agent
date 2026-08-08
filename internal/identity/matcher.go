package identity

import (
	"path/filepath"
	"strings"
)

// Matcher identifies applications against a known-AI catalog.
type Matcher struct {
	catalog *Catalog
}

// NewMatcher returns a matcher bound to catalog (DefaultCatalog if nil).
func NewMatcher(catalog *Catalog) *Matcher {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	return &Matcher{catalog: catalog}
}

// Identify selects catalog candidates by name/path, then verifies with strong
// identity evidence. Names and paths alone never produce VERIFIED.
func (m *Matcher) Identify(app ApplicationIdentity) IdentificationResult {
	if m == nil || m.catalog == nil {
		return IdentificationResult{Confidence: ConfidenceUnknown}
	}

	candidates := m.candidates(app)
	if len(candidates) == 0 {
		return IdentificationResult{
			Confidence: ConfidenceUnknown,
			Failed:     []EvidenceKey{EvidenceCandidateName, EvidenceCandidatePath},
		}
	}

	best := IdentificationResult{Confidence: ConfidenceUnknown}
	for i := range candidates {
		p := candidates[i]
		res := scoreAgainst(app, &p)
		if confidenceRank(res.Confidence) > confidenceRank(best.Confidence) {
			best = res
			// Heap-allocate so Product outlives this function.
			cp := p
			best.Product = &cp
		}
	}
	return best
}

func (m *Matcher) candidates(app ApplicationIdentity) []KnownAIProduct {
	var out []KnownAIProduct
	for _, p := range m.catalog.Products() {
		nameHit := nameMatches(app, p)
		pathHit := pathMatches(app, p)
		if nameHit || pathHit {
			out = append(out, p)
		}
	}
	return out
}

func nameMatches(app ApplicationIdentity, p KnownAIProduct) bool {
	exe := strings.TrimSpace(app.Executable)
	if exe == "" && app.ExecutablePath != "" {
		exe = filepath.Base(app.ExecutablePath)
	}
	if exe == "" && app.InvocationPath != "" {
		exe = filepath.Base(app.InvocationPath)
	}
	if exe == "" && app.ResolvedPath != "" {
		exe = filepath.Base(app.ResolvedPath)
	}
	if exe == "" && app.Path != "" {
		// Fall back to bundle display name stem (Cursor.app → Cursor).
		base := filepath.Base(app.Path)
		exe = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if exe == "" {
		return false
	}
	for _, n := range p.CandidateNames {
		if !strings.EqualFold(exe, n) {
			continue
		}
		// Bare lowercase commands must not EqualFold-match Title-Case GUI
		// candidates (e.g. "claude" vs Claude Desktop "Claude").
		if len(p.BundleIDs) > 0 && exe != n && !looksLikeAppBundle(app) {
			continue
		}
		return true
	}
	// ExecutableNames are exact (case-sensitive) so CLI "claude" does not
	// collide with GUI "Claude".
	for _, n := range p.ExecutableNames {
		if exe == n {
			return true
		}
	}
	return false
}

func looksLikeAppBundle(app ApplicationIdentity) bool {
	path := app.Path
	if path == "" {
		path = app.ExecutablePath
	}
	return strings.Contains(strings.ToLower(path), ".app/") || strings.HasSuffix(strings.ToLower(path), ".app")
}

func pathMatches(app ApplicationIdentity, p KnownAIProduct) bool {
	paths := []string{app.Path, app.ExecutablePath}
	for _, got := range paths {
		got = strings.TrimSpace(got)
		if got == "" {
			continue
		}
		for _, cand := range p.CandidatePaths {
			cand = expandHome(cand)
			if cand == "" {
				continue
			}
			if pathEqualOrWithin(got, cand) {
				return true
			}
		}
	}
	return false
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		// Candidate path matching uses literal prefix comparison against
		// already-expanded absolute paths when possible; leave ~ for
		// suffix/basename-style contains checks via pathEqualOrWithin.
		return p
	}
	return p
}

func pathEqualOrWithin(got, candidate string) bool {
	got = filepath.Clean(got)
	candidate = filepath.Clean(candidate)
	if strings.HasPrefix(candidate, "~/") {
		// Match by path suffix: "/Users/x/Applications/Cursor.app" ends with "/Applications/Cursor.app"
		suf := candidate[1:] // "/Applications/Cursor.app"
		if got == suf || strings.HasSuffix(got, suf) {
			return true
		}
		return strings.Contains(strings.ToLower(got), strings.ToLower(filepath.Base(candidate)))
	}
	if strings.EqualFold(got, candidate) {
		return true
	}
	// Executable inside the candidate .app bundle.
	prefix := candidate + string(filepath.Separator)
	return strings.HasPrefix(got, prefix)
}

func scoreAgainst(app ApplicationIdentity, p *KnownAIProduct) IdentificationResult {
	res := IdentificationResult{
		Product:    p,
		Confidence: ConfidenceLow, // candidate matched by name/path at minimum
	}

	cliStyle := p.Category == ProductCategoryCLIAgent || len(p.BundleIDs) == 0

	// Candidate evidence (discovery only).
	if nameMatches(app, *p) {
		res.Matched = append(res.Matched, EvidenceCandidateName)
		if cliStyle {
			res.Matched = append(res.Matched, EvidenceCommand)
		}
	} else {
		res.Failed = append(res.Failed, EvidenceCandidateName)
	}
	if pathMatches(app, *p) {
		res.Matched = append(res.Matched, EvidenceCandidatePath)
	} else {
		res.Failed = append(res.Failed, EvidenceCandidatePath)
	}

	bundleOK := evalStringEvidence(app.BundleID, p.BundleIDs, EvidenceBundleID, &res)
	signOK := evalStringEvidence(app.SigningIdentifier, p.SigningIdentifiers, EvidenceSigningIdentifier, &res)
	teamOK := evalStringEvidence(app.TeamID, p.TeamIDs, EvidenceTeamID, &res)
	sigOK := evalSignature(app, &res)
	hashOK := evalHash(app, p, &res)

	var pkgMgrOK, pkgIDOK, entryOK, provOK bool
	if len(p.PackageManagers) > 0 || len(p.PackageIdentifiers) > 0 || len(p.EntryPoints) > 0 ||
		app.PackageManager != "" || app.PackageIdentifier != "" || p.Category == ProductCategoryCLIAgent {
		pkgMgrOK = evalStringEvidence(app.PackageManager, p.PackageManagers, EvidencePackageManager, &res)
		pkgIDOK = evalStringEvidence(app.PackageIdentifier, p.PackageIdentifiers, EvidencePackageIdentity, &res)
		entryOK = evalStringEvidence(entryPointOf(app), p.EntryPoints, EvidenceEntryPoint, &res)
		provOK = packageProvenanceOK(app, p, &res)
	}

	// Vacuous catalog lists must not inflate confidence for CLI / non-bundle products.
	if cliStyle {
		if len(p.BundleIDs) == 0 {
			bundleOK = false
		}
		if len(p.SigningIdentifiers) == 0 {
			signOK = false
		}
		if len(p.TeamIDs) == 0 {
			teamOK = false
		}
		if len(p.ExpectedHashes) == 0 {
			hashOK = false
		}
		if !app.SignatureChecked {
			sigOK = false
		}
		if len(p.PackageManagers) == 0 {
			pkgMgrOK = false
		}
		if len(p.PackageIdentifiers) == 0 {
			pkgIDOK = false
		}
		if len(p.EntryPoints) == 0 {
			entryOK = false
		}
	}

	strong := 0
	for _, ok := range []bool{bundleOK, signOK, teamOK, sigOK} {
		if ok {
			strong++
		}
	}
	if hashOK && len(p.ExpectedHashes) > 0 {
		strong++
	}
	for _, ok := range []bool{pkgMgrOK, pkgIDOK, provOK, entryOK} {
		if ok {
			strong++
		}
	}

	switch {
	case !cliStyle && bundleOK && signOK && teamOK && sigOK && hashSatisfied(app, p, hashOK):
		res.Confidence = ConfidenceVerified
	case !cliStyle && bundleOK && sigOK && (signOK || teamOK):
		res.Confidence = ConfidenceHigh
	case !cliStyle && bundleOK && (signOK || teamOK):
		res.Confidence = ConfidenceMedium
	case cliStyle && pkgIDOK && provOK && (entryOK || signOK || sigOK):
		// VERIFIED CLI requires catalog package IDs (non-empty) plus provenance.
		res.Confidence = ConfidenceVerified
	case cliStyle && pkgIDOK && (provOK || pkgMgrOK):
		res.Confidence = ConfidenceHigh
	case cliStyle && (pkgIDOK || pkgMgrOK || provOK) && strong >= 1:
		res.Confidence = ConfidenceMedium
	case strong >= 2:
		res.Confidence = ConfidenceMedium
	case strong == 1:
		res.Confidence = ConfidenceLow
	default:
		// Name/path candidate only — still LOW, never VERIFIED.
		res.Confidence = ConfidenceLow
	}
	return res
}

func entryPointOf(app ApplicationIdentity) string {
	if app.EntryPoint != "" {
		return app.EntryPoint
	}
	if app.ResolvedPath != "" {
		return filepath.Base(app.ResolvedPath)
	}
	return ""
}

func packageProvenanceOK(app ApplicationIdentity, p *KnownAIProduct, res *IdentificationResult) bool {
	if app.PackageManager == "" && app.PackageIdentifier == "" {
		if len(p.PackageIdentifiers) > 0 || len(p.PackageManagers) > 0 {
			res.Failed = append(res.Failed, EvidencePackageProvenance)
		}
		return false
	}
	// Observed provenance present; identity match evaluated separately.
	res.Matched = append(res.Matched, EvidencePackageProvenance)
	return true
}

func hashSatisfied(app ApplicationIdentity, p *KnownAIProduct, hashOK bool) bool {
	if len(p.ExpectedHashes) == 0 {
		// Hash not required when catalog has no pinned digests.
		return true
	}
	return hashOK
}

func evalStringEvidence(got string, want []string, key EvidenceKey, res *IdentificationResult) bool {
	got = strings.TrimSpace(got)
	if len(want) == 0 {
		// Catalog does not require this factor.
		return true
	}
	if got == "" {
		res.Failed = append(res.Failed, key)
		return false
	}
	for _, w := range want {
		if strings.EqualFold(got, w) {
			res.Matched = append(res.Matched, key)
			return true
		}
	}
	res.Failed = append(res.Failed, key)
	return false
}

func evalSignature(app ApplicationIdentity, res *IdentificationResult) bool {
	if !app.SignatureChecked {
		res.Failed = append(res.Failed, EvidenceSignatureValid)
		return false
	}
	if app.SignatureValid {
		res.Matched = append(res.Matched, EvidenceSignatureValid)
		return true
	}
	res.Failed = append(res.Failed, EvidenceSignatureValid)
	return false
}

func evalHash(app ApplicationIdentity, p *KnownAIProduct, res *IdentificationResult) bool {
	if len(p.ExpectedHashes) == 0 {
		return false // not applicable; hashSatisfied treats as optional
	}
	got := strings.TrimSpace(strings.ToLower(app.SHA256))
	if got == "" {
		res.Failed = append(res.Failed, EvidenceSHA256)
		return false
	}
	for _, w := range p.ExpectedHashes {
		if got == strings.ToLower(strings.TrimSpace(w)) {
			res.Matched = append(res.Matched, EvidenceSHA256)
			return true
		}
	}
	res.Failed = append(res.Failed, EvidenceSHA256)
	return false
}

func confidenceRank(c Confidence) int {
	switch c {
	case ConfidenceVerified:
		return 5
	case ConfidenceHigh:
		return 4
	case ConfidenceMedium:
		return 3
	case ConfidenceLow:
		return 2
	default:
		return 1
	}
}
