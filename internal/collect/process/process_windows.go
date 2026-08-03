//go:build windows

package process

import (
	"os/exec"
)

var listProcesses = func() ([]processRow, error) {
	script := `Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,Name,ExecutablePath,CommandLine | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	return parseWinProcessJSON(string(out))
}
