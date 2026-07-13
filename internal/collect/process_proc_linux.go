//go:build linux

package collect

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
		User:       procOwner(filepath.Join("/proc", proc, "status")),
	}
	return row, true
}

func parseProcStat(stat []byte) (ppid int, comm string, ok bool) {
	fields := strings.Fields(string(stat))
	if len(fields) < 4 {
		return 0, "", false
	}
	open := strings.Index(string(stat), "(")
	close := strings.LastIndex(string(stat), ")")
	if open < 0 || close <= open {
		return 0, "", false
	}
	comm = string(stat[open+1 : close])
	ppid, err := strconv.Atoi(fields[3])
	if err != nil {
		return 0, "", false
	}
	return ppid, comm, true
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
