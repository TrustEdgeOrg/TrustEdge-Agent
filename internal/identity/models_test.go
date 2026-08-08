package identity

import "testing"

func TestConfidenceConstants(t *testing.T) {
	vals := []Confidence{
		ConfidenceVerified,
		ConfidenceHigh,
		ConfidenceMedium,
		ConfidenceLow,
		ConfidenceUnknown,
	}
	want := []string{"VERIFIED", "HIGH", "MEDIUM", "LOW", "UNKNOWN"}
	for i, v := range vals {
		if string(v) != want[i] {
			t.Fatalf("Confidence[%d]=%q want %q", i, v, want[i])
		}
	}
}

func TestIdentificationResultZeroValue(t *testing.T) {
	var r IdentificationResult
	if r.Product != nil {
		t.Fatal("Product should be nil")
	}
	if r.Installed || r.Running {
		t.Fatal("Installed/Running should be false by default")
	}
	if r.Confidence != "" {
		t.Fatalf("zero Confidence=%q", r.Confidence)
	}
}

func TestApplicationIdentityFields(t *testing.T) {
	id := ApplicationIdentity{
		Path:              "/Applications/Cursor.app",
		BundleID:          "com.example.app",
		Version:           "1.0.0",
		Executable:        "Cursor",
		SigningIdentifier: "com.example.app",
		TeamID:            "ABCD123456",
		SignatureValid:    true,
		SignatureChecked:  true,
	}
	if id.Path == "" || id.BundleID == "" {
		t.Fatal("expected identity fields set")
	}
	if !id.SignatureValid || !id.SignatureChecked {
		t.Fatal("expected signature flags set")
	}
}

func TestKnownAIProductAndEvidenceKeys(t *testing.T) {
	p := KnownAIProduct{
		ID:           "example",
		Name:         "Example",
		Vendor:       "Vendor",
		Category:     ProductCategoryCodeEditor,
		BundleIDs:    []string{"com.example"},
		TeamIDs:      []string{"TEAMID"},
		CandidateNames: []string{"Example"},
	}
	if p.Category != ProductCategoryCodeEditor {
		t.Fatalf("Category=%q", p.Category)
	}
	keys := []EvidenceKey{
		EvidenceBundleID,
		EvidenceSigningIdentifier,
		EvidenceTeamID,
		EvidenceSignatureValid,
		EvidenceSHA256,
		EvidenceCandidateName,
		EvidenceCandidatePath,
		EvidenceCommand,
		EvidencePackageManager,
		EvidencePackageIdentity,
		EvidencePackageProvenance,
		EvidenceEntryPoint,
		EvidenceInvocationPath,
	}
	if len(keys) != 13 {
		t.Fatalf("unexpected evidence key count %d", len(keys))
	}
	if ProductCategoryCLIAgent != "cli_agent" {
		t.Fatalf("CLI category=%q", ProductCategoryCLIAgent)
	}
}
