//go:build windows

package collect

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type winProcess struct {
	PID        int    `json:"ProcessId"`
	PPID       int    `json:"ParentProcessId"`
	Name       string `json:"Name"`
	Executable string `json:"ExecutablePath"`
	Cmdline    string `json:"CommandLine"`
}

var listProcesses = func() ([]processRow, error) {
	script := `Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,Name,ExecutablePath,CommandLine | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	return parseWinProcessJSON(string(out))
}

func parseWinProcessJSON(text string) ([]processRow, error) {
	text = strings.TrimSpace(text)
	if text == "" || text == "null" {
		return nil, nil
	}
	var rows []winProcess
	if text[0] == '{' {
		var one winProcess
		if err := json.Unmarshal([]byte(text), &one); err != nil {
			return nil, err
		}
		rows = []winProcess{one}
	} else {
		if err := json.Unmarshal([]byte(text), &rows); err != nil {
			return nil, err
		}
	}
	result := make([]processRow, 0, len(rows))
	for _, row := range rows {
		if row.PID <= 0 {
			continue
		}
		comm := strings.TrimSpace(row.Name)
		exe := strings.TrimSpace(row.Executable)
		cmdline := truncateCmdline(strings.TrimSpace(row.Cmdline))
		if comm == "" {
			comm = exe
		}
		result = append(result, processRow{
			PID:        row.PID,
			PPID:       row.PPID,
			User:       "",
			Comm:       comm,
			Executable: exe,
			Cmdline:    cmdline,
		})
	}
	return result, nil
}
