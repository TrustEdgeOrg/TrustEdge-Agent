//go:build darwin

package process

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
)

func parsePSOutput(out []byte) []processRow {
	var rows []processRow
	scanner := bufio.NewScanner(bytes.NewReader(out))
	// Long command lines can exceed the default token size.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
		cmdline := truncateCmdline(strings.Join(fields[3:], " "))
		exe := fields[3]
		comm := filepath.Base(exe)
		rows = append(rows, processRow{
			PID:        pid,
			PPID:       ppid,
			User:       user,
			Comm:       comm,
			Executable: exe,
			Cmdline:    cmdline,
		})
	}
	return rows
}
