//go:build !darwin

package apps

type stubSigner struct{}

func newPlatformSigner() Signer {
	return stubSigner{}
}

func (stubSigner) Extract(path string) (SigningInfo, error) {
	_ = path
	return SigningInfo{}, nil
}

func (stubSigner) Validate(path string) (bool, error) {
	_ = path
	return false, nil
}
