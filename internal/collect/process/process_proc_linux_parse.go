//go:build linux

package process

import (
	"strconv"
	"strings"
)

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
