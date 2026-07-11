//go:build !darwin && !linux

package collect

var listProcesses = func() ([]processRow, error) {
	return nil, nil
}
