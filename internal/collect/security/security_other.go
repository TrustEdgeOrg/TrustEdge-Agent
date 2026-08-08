//go:build !windows && !darwin

package security

var listSecurityArtifacts = func() ([]SecurityArtifact, error) {
	return nil, nil
}
