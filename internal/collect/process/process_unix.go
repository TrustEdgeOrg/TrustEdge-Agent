//go:build darwin

package process

import (
	"os/exec"
)

var listProcesses = func() ([]processRow, error) {
	// args= must be last so remaining fields reconstruct the command line.
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,user=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parsePSOutput(out), nil
}
