//go:build !darwin

package apps

// emptySigner leaves signing unresolved on non-macOS builds.
type emptySigner struct{}

func newPlatformSigner() Signer {
	return emptySigner{}
}

func (emptySigner) Extract(path string) (SigningInfo, error) {
	_ = path
	return SigningInfo{}, nil
}

func (emptySigner) Validate(path string) (bool, error) {
	_ = path
	return false, nil
}
