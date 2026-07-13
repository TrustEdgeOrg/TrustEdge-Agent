//go:build darwin || linux

package collect

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

var listProcesses = func() ([]processRow, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,user=,comm=").Output()
	if err != nil {
		return nil, err
	}
	return parsePSOutput(out), nil
}

func parsePSOutput(out []byte) []processRow {
	var rows []processRow
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		user := fields[2]
		comm := fields[3]
		rows = append(rows, processRow{
			PID:        pid,
			PPID:       ppid,
			User:       user,
			Comm:       comm,
			Executable: comm,
		})
	}
	return rows
}
