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

	var base string
	switch level {
	case ConfidenceVerified:
		base = "Strong identity match: bundle, signing, team, and signature all checked out"
	case ConfidenceHigh:
		base = "Strong match: bundle ID and code signature verified, with signing or team evidence"
	case ConfidenceMedium:
		base = "Partial cryptographic identity match; some strong factors are missing"
	case ConfidenceLow:
		if containsEvidence(res.Failed, EvidenceSignatureValid) ||
			containsEvidence(res.Failed, EvidenceSigningIdentifier) ||
			containsEvidence(res.Failed, EvidenceTeamID) {
			base = "Recognized mainly by name/path or bundle ID; code-signing evidence is missing"
		} else {
			base = "Recognized mainly by name or install path; strong identity checks did not pass"
		}
	case ConfidenceUnknown:
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
