//go:build darwin && cgo

package apps

import (
	"os"
	"testing"
)

func TestLiveCursorSigningExtraction(t *testing.T) {
	const path = "/Applications/Cursor.app"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Cursor.app not installed")
	}
	info, err := ExtractAndValidate(NewSigner(), path)
	if err != nil {
		t.Fatal(err)
	}
	if info.SigningIdentifier != "com.todesktop.230313mzl4w4u92" {
		t.Fatalf("SigningIdentifier=%q", info.SigningIdentifier)
	}
	if info.TeamID != "VDXQ22DGB9" {
		t.Fatalf("TeamID=%q", info.TeamID)
	}
	if !info.SignatureChecked || !info.SignatureValid {
		t.Fatalf("signature flags checked=%v valid=%v", info.SignatureChecked, info.SignatureValid)
	}
	if info.CertificateSubject == "" {
		t.Fatal("expected certificate subject")
	}
}
