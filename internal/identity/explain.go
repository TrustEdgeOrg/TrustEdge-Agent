package identity

import (
	"fmt"
	"strings"
)

// EvidenceLabel is a human-readable name for an evidence factor.
func EvidenceLabel(key EvidenceKey) string {
	switch key {
	case EvidenceCandidateName:
		return "app name"
	case EvidenceCandidatePath:
		return "install path"
	case EvidenceBundleID:
		return "bundle ID"
	case EvidenceSigningIdentifier:
		return "code signing ID"
	case EvidenceTeamID:
		return "Apple Team ID"
	case EvidenceSignatureValid:
		return "valid code signature"
	case EvidenceSHA256:
		return "binary hash"
	case EvidenceCommand:
		return "command name"
	case EvidencePackageManager:
		return "package manager"
	case EvidencePackageIdentity:
		return "package identity"
	case EvidencePackageProvenance:
		return "package provenance"
	case EvidenceEntryPoint:
		return "entry point"
	case EvidenceInvocationPath:
		return "invocation path"
	case EvidenceListener:
		return "network listener"
	case EvidenceListenerExposure:
		return "listener exposure"
	case EvidenceRuntimeFingerprint:
		return "runtime fingerprint"
	case EvidenceModelArtifact:
		return "model artifact"
	case EvidenceLocalClient:
		return "local client"
	default:
		return strings.ReplaceAll(string(key), "_", " ")
	}
}

func evidenceLabels(keys []EvidenceKey) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, EvidenceLabel(k))
	}
	return out
}

// ExplainConfidence returns an operator-facing reason for an identification result.
// This text is intended to be shown in the dashboard confidence UI.
func ExplainConfidence(res IdentificationResult) string {
	level := res.Confidence
	if level == "" {
		level = ConfidenceUnknown
	}
	matched := evidenceLabels(res.Matched)
	failed := evidenceLabels(res.Failed)

	runtime := res.Product != nil && res.Product.Category == ProductCategoryLocalModelRuntime
	cli := res.Product != nil && (res.Product.Category == ProductCategoryCLIAgent || len(res.Product.BundleIDs) == 0) && !runtime

	var base string
	switch {
	case level == ConfidenceVerified && runtime:
		base = "Strong local model runtime identity: package identity and provenance verified"
	case level == ConfidenceVerified && cli:
		base = "Strong CLI identity match: package identity and provenance verified"
	case level == ConfidenceVerified:
		base = "Strong identity match: bundle, signing, team, and signature all checked out"
	case level == ConfidenceHigh && runtime:
		base = "Strong local model runtime match: package identity aligned with observed provenance"
	case level == ConfidenceHigh && cli:
		base = "Strong CLI match: package identity aligned with observed provenance"
	case level == ConfidenceHigh:
		base = "Strong match: bundle ID and code signature verified, with signing or team evidence"
	case level == ConfidenceMedium && runtime:
		base = "Partial local model runtime package evidence; catalog pins incomplete"
	case level == ConfidenceMedium && cli:
		base = "Partial CLI package evidence; catalog package pins or entry-point checks incomplete"
	case level == ConfidenceMedium:
		base = "Partial cryptographic identity match; some strong factors are missing"
	case level == ConfidenceLow && runtime:
		base = "Recognized mainly by runtime command name; package identity is unresolved or unmatched; not classified as an AI agent"
	case level == ConfidenceLow && cli:
		base = "Recognized mainly by command name; package identity is unresolved or unmatched"
	case level == ConfidenceLow:
		if containsEvidence(res.Failed, EvidenceSignatureValid) ||
			containsEvidence(res.Failed, EvidenceSigningIdentifier) ||
			containsEvidence(res.Failed, EvidenceTeamID) {
			base = "Recognized mainly by name/path or bundle ID; code-signing evidence is missing"
		} else {
			base = "Recognized mainly by name or install path; strong identity checks did not pass"
		}
	case level == ConfidenceUnknown:
		base = "Could not confidently identify this application"
	default:
		base = "Identification confidence is based on catalog evidence"
	}

	parts := []string{base}
	if len(matched) > 0 {
		parts = append(parts, "matched: "+strings.Join(matched, ", "))
	}
	if len(failed) > 0 {
		parts = append(parts, "missing: "+strings.Join(failed, ", "))
	}
	return fmt.Sprintf("%s. %s.", string(level), strings.Join(parts, "; "))
}

func containsEvidence(keys []EvidenceKey, want EvidenceKey) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
