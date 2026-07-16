//go:build !windows && !darwin

package collect

var listSecurityArtifacts = func() ([]SecurityArtifact, error) {
	return nil, nil
}
