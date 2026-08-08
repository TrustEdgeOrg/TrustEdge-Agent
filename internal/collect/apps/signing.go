package apps

import "github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"

// SigningInfo holds code-signing identity extracted from a path.
// Extraction and validation are separate: SignatureValid is only meaningful
// when SignatureChecked is true.
type SigningInfo struct {
	SigningIdentifier  string
	TeamID             string
	CertificateSubject string
	SignatureValid     bool
	SignatureChecked   bool
}

// Signer extracts and validates macOS code-signing identity.
type Signer interface {
	// Extract returns signing identifier, Team ID, and certificate metadata
	// without treating signature validity as required for the return.
	Extract(path string) (SigningInfo, error)
	// Validate checks static code signature validity for path.
	Validate(path string) (valid bool, err error)
}

// NewSigner returns a platform signer. On non-macOS or without CGO it returns
// a no-op signer that leaves signing fields unresolved.
func NewSigner() Signer {
	return newPlatformSigner()
}

// ApplySigning merges signing info into an ApplicationIdentity.
func ApplySigning(id *identity.ApplicationIdentity, info SigningInfo) {
	if id == nil {
		return
	}
	if info.SigningIdentifier != "" {
		id.SigningIdentifier = info.SigningIdentifier
	}
	if info.TeamID != "" {
		id.TeamID = info.TeamID
	}
	if info.CertificateSubject != "" {
		id.CertificateSubject = info.CertificateSubject
	}
	id.SignatureValid = info.SignatureValid
	id.SignatureChecked = info.SignatureChecked
}

// ExtractAndValidate runs Extract then Validate separately and records both results.
func ExtractAndValidate(s Signer, path string) (SigningInfo, error) {
	info, err := s.Extract(path)
	if err != nil {
		return info, err
	}
	valid, verr := s.Validate(path)
	info.SignatureChecked = true
	if verr != nil {
		info.SignatureValid = false
		return info, verr
	}
	info.SignatureValid = valid
	return info, nil
}
