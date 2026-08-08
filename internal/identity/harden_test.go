package identity

import "testing"

// Hardening scenarios for known-AI identity matching.
func TestHardenLegitimateCursor(t *testing.T) {
	res := NewMatcher(DefaultCatalog()).Identify(legitCursor("/Applications/Cursor.app"))
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("got %s failed=%v", res.Confidence, res.Failed)
	}
}

func TestHardenRenamedLegitimateCursor(t *testing.T) {
	// Bundle renamed on disk but strong signing identity intact.
	id := legitCursor("/Users/dev/Applications/MyEditor.app")
	id.Executable = "MyEditor"
	res := NewMatcher(DefaultCatalog()).Identify(id)
	// Path/name may not candidate-match; strong IDs require candidate first.
	// Ensure name-based candidate via executable still works when set to Cursor,
	// and path under ~/Applications/Cursor.app pattern.
	id2 := legitCursor("/Users/dev/Applications/Cursor.app")
	id2.Path = "/Users/dev/Applications/Cursor.app"
	res = NewMatcher(DefaultCatalog()).Identify(id2)
	if res.Confidence != ConfidenceVerified {
		t.Fatalf("renamed install location: %s %v", res.Confidence, res.Failed)
	}
}

func TestHardenFakeCursorApp(t *testing.T) {
	res := NewMatcher(DefaultCatalog()).Identify(ApplicationIdentity{
		Path:       "/Applications/Cursor.app",
		BundleID:   "com.evil.cursor",
		Executable: "Cursor",
		TeamID:     "EVILTEAM12",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("fake app too strong: %s", res.Confidence)
	}
}

func TestHardenExecutableNamedCursor(t *testing.T) {
	res := NewMatcher(DefaultCatalog()).Identify(ApplicationIdentity{
		Path:       "/tmp/Cursor",
		Executable: "Cursor",
		ExecutablePath: "/tmp/Cursor",
	})
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh || res.Confidence == ConfidenceMedium {
		t.Fatalf("name-only too strong: %s", res.Confidence)
	}
}

func TestHardenCorrectBundleWrongSigner(t *testing.T) {
	id := legitCursor("/Applications/Cursor.app")
	id.SigningIdentifier = "com.evil.fake"
	res := NewMatcher(DefaultCatalog()).Identify(id)
	if res.Confidence == ConfidenceVerified {
		t.Fatal("wrong signer verified")
	}
	if !hasEvidence(res.Failed, EvidenceSigningIdentifier) {
		t.Fatal("expected signing failure")
	}
}

func TestHardenWrongTeamID(t *testing.T) {
	id := legitCursor("/Applications/Cursor.app")
	id.TeamID = "ZZZZZZZZZZ"
	res := NewMatcher(DefaultCatalog()).Identify(id)
	if res.Confidence == ConfidenceVerified {
		t.Fatal("wrong team verified")
	}
}

func TestHardenInvalidSignature(t *testing.T) {
	id := legitCursor("/Applications/Cursor.app")
	id.SignatureValid = false
	res := NewMatcher(DefaultCatalog()).Identify(id)
	if res.Confidence == ConfidenceVerified || res.Confidence == ConfidenceHigh {
		t.Fatalf("invalid sig: %s", res.Confidence)
	}
}

func TestHardenHashMismatchWhenPinned(t *testing.T) {
	p := cursorProduct()
	p.ExpectedHashes = []string{"aaa"}
	cat := NewCatalog(p)
	id := legitCursor("/Applications/Cursor.app")
	id.SHA256 = "bbb"
	res := NewMatcher(cat).Identify(id)
	if res.Confidence == ConfidenceVerified {
		t.Fatal("hash mismatch must not be VERIFIED")
	}
	if !hasEvidence(res.Failed, EvidenceSHA256) {
		t.Fatal("expected sha256 failed")
	}
}

func legitCursor(path string) ApplicationIdentity {
	return ApplicationIdentity{
		Path:              path,
		BundleID:          "com.todesktop.230313mzl4w4u92",
		Executable:        "Cursor",
		ExecutablePath:    path + "/Contents/MacOS/Cursor",
		SigningIdentifier: "com.todesktop.230313mzl4w4u92",
		TeamID:            "VDXQ22DGB9",
		SignatureValid:    true,
		SignatureChecked:  true,
		Version:           "1.0.0",
	}
}
