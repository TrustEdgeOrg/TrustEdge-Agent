package identity

import (
	"strings"
	"testing"
)

func TestExplainConfidenceLowSigningMissing(t *testing.T) {
	res := IdentificationResult{
		Confidence: ConfidenceLow,
		Matched:    []EvidenceKey{EvidenceCandidateName, EvidenceCandidatePath, EvidenceBundleID},
		Failed:     []EvidenceKey{EvidenceSigningIdentifier, EvidenceTeamID, EvidenceSignatureValid},
	}
	got := ExplainConfidence(res)
	low := strings.ToLower(got)
	for _, part := range []string{"low", "code-signing", "matched:", "missing:", "bundle id", "valid code signature"} {
		if !strings.Contains(low, part) {
			t.Fatalf("missing %q in explanation: %q", part, got)
		}
	}
}

func TestExplainConfidenceVerified(t *testing.T) {
	res := IdentificationResult{
		Confidence: ConfidenceVerified,
		Matched: []EvidenceKey{
			EvidenceBundleID,
			EvidenceSigningIdentifier,
			EvidenceTeamID,
			EvidenceSignatureValid,
		},
	}
	got := ExplainConfidence(res)
	if !strings.Contains(got, "VERIFIED") || !strings.Contains(got, "Strong identity") {
		t.Fatalf("unexpected explanation: %q", got)
	}
}
