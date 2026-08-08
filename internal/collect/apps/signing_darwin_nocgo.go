//go:build darwin && !cgo

package apps

// nocgoSigner leaves signing unresolved when the agent is built without CGO.
type nocgoSigner struct{}

func newPlatformSigner() Signer {
	return nocgoSigner{}
}

func (nocgoSigner) Extract(path string) (SigningInfo, error) {
	_ = path
	return SigningInfo{}, nil
}

func (nocgoSigner) Validate(path string) (bool, error) {
	_ = path
	return false, nil
}
