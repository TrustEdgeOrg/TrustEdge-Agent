//go:build linux

package process

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

func processRowFromPID(pid int) (processRow, bool) {
	if pid <= 0 {
		return processRow{}, false
	}
	proc := strconv.Itoa(pid)
	stat, err := os.ReadFile(filepath.Join("/proc", proc, "stat"))
	if err != nil {
		return processRow{}, false
	}
	ppid, comm, ok := parseProcStat(stat)
	if !ok {
		return processRow{}, false
	}
	exe, _ := os.Readlink(filepath.Join("/proc", proc, "exe"))
	commName := comm
	if exe != "" {
		commName = filepath.Base(exe)
	}
	row := processRow{
		PID:        pid,
		PPID:       ppid,
		Comm:       commName,
		Executable: exe,
		Cmdline:    readProcCmdline(proc),
		User:       procOwner(filepath.Join("/proc", proc, "status")),
	}
	return row, true
}

func readProcCmdline(proc string) string {
	data, err := os.ReadFile(filepath.Join("/proc", proc, "cmdline"))
	if err != nil {
		return ""
	}
	return joinNullSeparatedCmdline(data)
}

var listProcesses = func() ([]processRow, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	rows := make([]processRow, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		row, ok := processRowFromPID(pid)
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func procOwner(statusPath string) string {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return ""
		}
		uid, err := strconv.Atoi(fields[1])
		if err != nil {
			return ""
		}
		u, err := user.LookupId(fmt.Sprintf("%d", uid))
		if err != nil {
			return ""
		}
		return u.Username
	}
	return ""
}
