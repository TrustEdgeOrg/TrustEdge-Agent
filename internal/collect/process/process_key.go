package process

// ProcessKey uniquely identifies a process instance to avoid PID reuse bleed.
// StartTimeUnixNano is 0 when the platform does not provide a start time.
type ProcessKey struct {
	PID               int
	StartTimeUnixNano int64
}

// Key returns the process instance key for this snapshot row.
func (p ProcessInfo) Key() ProcessKey {
	return ProcessKey{PID: p.PID, StartTimeUnixNano: p.StartTimeUnixNano}
}
