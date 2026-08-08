package identity

import "testing"

func TestIdentifyVerifiedCursor(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Applications/Cursor.app",
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
		SignatureChecked:  true,
	})
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("Confidence=%s matched=%v failed=%v", res.Confidence, res.Matched, res.Failed)
	}
	if res.Product == nil || res.Product.ID != ProductCursorID {
		t.Fatalf("Product=%v", res.Product)
	}
}

func TestIdentifyVerifiedClaude(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Applications/Claude.app",
		BundleID:          "com.anthropic.claudefordesktop",
		Executable:        "Claude",
		SigningIdentifier: "com.anthropic.claudefordesktop",
		TeamID:            "Q6L2SF6YDW",
		SignatureValid:    true,
		SignatureChecked:  true,
	})
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("Confidence=%s matched=%v failed=%v", res.Confidence, res.Matched, res.Failed)
	}
	if res.Product == nil || res.Product.ID != ProductClaudeID {
		t.Fatalf("Product=%v", res.Product)
	}
}

func TestIdentifyClaudeNameAloneNeverVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:       "/tmp/evil/Claude.app",
		Executable: "Claude",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("name/path alone must not be strong: %s", res.Confidence)
	}
	if res.Product == nil || res.Product.ID != ProductClaudeID {
		t.Fatalf("expected claude candidate, got %v", res.Product)
	}
}

func TestIdentifyNameAloneNeverVerified(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:       "/tmp/evil/Cursor.app",
		Executable: "Cursor",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("name/path alone must not be strong: %s", res.Confidence)
	}
	if res.Confidence != ConfidenceLow {
		t.Fatalf("want LOW candidate, got %s", res.Confidence)
	}
	if !hasEvidence(res.Matched, EvidenceCandidateName) {
		t.Fatal("expected candidate_name matched")
	}
	if !hasEvidence(res.Failed, EvidenceBundleID) {
		t.Fatal("expected bundle_id failed")
	}
}

func TestIdentifyWrongTeamID(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Applications/Cursor.app",
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "AAAAAAAAAA",
		SignatureValid:    true,
		SignatureChecked:  true,
	})
	if res.Confidence == ConfidenceVerified {
		t.Fatal("wrong Team ID must not be VERIFIED")
	}
	if !hasEvidence(res.Failed, EvidenceTeamID) {
		t.Fatal("expected team_id failed")
	}
}

func TestIdentifyWrongSigner(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Applications/Cursor.app",
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		SigningIdentifier: "com.evil.fake",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
		SignatureChecked:  true,
	})
	if res.Confidence == ConfidenceVerified {
		t.Fatal("wrong signing identifier must not be VERIFIED")
	}
	if !hasEvidence(res.Failed, EvidenceSigningIdentifier) {
		t.Fatal("expected signing_identifier failed")
	}
}

func TestIdentifyInvalidSignature(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Applications/Cursor.app",
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    false,
		SignatureChecked:  true,
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("invalid signature too strong: %s", res.Confidence)
	}
	if !hasEvidence(res.Failed, EvidenceSignatureValid) {
		t.Fatal("expected signature_valid failed")
	}
}

func TestIdentifyUnknownApp(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:       "/Applications/Safari.app",
		Executable: "Safari",
		BundleID:   "com.apple.Safari",
	})
	if res.Confidence != ConfidenceUnknown {
		t.Fatalf("want UNKNOWN, got %s", res.Confidence)
	}
	if res.Product != nil {
		t.Fatal("expected nil product")
	}
}

func TestIdentifyHomeApplicationsPath(t *testing.T) {
	m := NewMatcher(DefaultCatalog())
	res := m.Identify(ApplicationIdentity{
		Path:              "/Users/dev/Applications/Cursor.app",
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
		SignatureChecked:  true,
	})
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("Confidence=%s failed=%v", res.Confidence, res.Failed)
	}
}

func hasEvidence(keys []EvidenceKey, want EvidenceKey) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
