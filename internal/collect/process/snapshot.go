package process

// ProcessInfo is a platform-neutral snapshot of a running process.
// Used by other collectors (e.g. known-AI correlation) without depending on
// unexported processRow.
type ProcessInfo struct {
	PID        int
	PPID       int
	User       string
	Comm       string
	Executable string
	Cmdline    string
}

// Snapshot returns the current process table via the platform listProcesses hook.
func Snapshot() ([]ProcessInfo, error) {
	rows, err := listProcesses()
	if err != nil {
		return nil, err
	}
	out := make([]ProcessInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProcessInfo{
			PID:        r.PID,
			PPID:       r.PPID,
			User:       r.User,
			Comm:       r.Comm,
			Executable: r.Executable,
			Cmdline:    r.Cmdline,
		})
	}
	return out, nil
}
