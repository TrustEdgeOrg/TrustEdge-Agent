//go:build !darwin && !linux && !windows

package collect

var listProcesses = func() ([]processRow, error) {
	return nil, nil
}
