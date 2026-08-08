package apps

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

type fakeSigner struct {
	info  SigningInfo
	valid bool
	err   error
}

func (f fakeSigner) Extract(path string) (SigningInfo, error) {
	_ = path
	return f.info, f.err
}

func (f fakeSigner) Validate(path string) (bool, error) {
	_ = path
	if f.err != nil {
		return false, f.err
	}
	// Prefer explicit valid; fall back to SignatureValid on info for test fixtures.
	return f.valid || f.info.SignatureValid, nil
}

func TestApplySigning(t *testing.T) {
	id := identity.ApplicationIdentity{Path: "/Applications/X.app"}
	ApplySigning(&id, SigningInfo{
		SigningIdentifier:  "com.example.x",
		TeamID:             "TEAMID1234",
		CertificateSubject: "Developer ID Application: Example",
		SignatureValid:     true,
		SignatureChecked:   true,
	})
	if id.SigningIdentifier != "com.example.x" || id.TeamID != "TEAMID1234" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if !id.SignatureValid || !id.SignatureChecked {
		t.Fatal("expected signature flags")
	}
}

func TestApplySigningNilSafe(t *testing.T) {
	ApplySigning(nil, SigningInfo{TeamID: "X"})
}

func TestFakeSignerSeparation(t *testing.T) {
	s := fakeSigner{
		info: SigningInfo{
			SigningIdentifier: "com.example",
			TeamID:            "ABCD123456",
		},
		valid: true,
	}
	info, err := s.Extract("/tmp/X.app")
	if err != nil {
		t.Fatal(err)
	}
	if info.SignatureChecked {
		t.Fatal("Extract must not set SignatureChecked; validation is separate")
	}
	valid, err := s.Validate("/tmp/X.app")
	if err != nil || !valid {
		t.Fatalf("Validate=%v err=%v", valid, err)
	}
}

func TestNewSignerDoesNotPanic(t *testing.T) {
	s := NewSigner()
	if s == nil {
		t.Fatal("nil signer")
	}
	// Without CGO or on non-macOS this returns empty info; with CGO it may
	// fail on a missing path — either outcome is acceptable here.
	_, _ = s.Extract("/nonexistent/TrustEdgeTest.app")
}
