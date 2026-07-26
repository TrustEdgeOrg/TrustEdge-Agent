//go:build !darwin && !linux && !windows

package process

var listProcesses = func() ([]processRow, error) {
	return nil, nil
}
