package collect

import (
	"bytes"
	"log"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

type processRow struct {
	PID        int
	PPID       int
	User       string
	Comm       string
	Executable string
	Cmdline    string
}

const (
	maxProcessEventsPerPoll = 100
	maxCmdlineBytes         = 4096
)

// truncateCmdline caps oversized command lines to keep event payloads bounded.
func truncateCmdline(s string) string {
	if len(s) <= maxCmdlineBytes {
		return s
	}
	return s[:maxCmdlineBytes] + "..."
}

// joinNullSeparatedCmdline turns /proc-style NUL-separated argv bytes into a space-joined string.
func joinNullSeparatedCmdline(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	parts := bytes.Split(data, []byte{0})
	args := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		args = append(args, string(p))
	}
	if len(args) == 0 {
		return ""
	}
	return truncateCmdline(strings.Join(args, " "))
}

// ProcessChange is a process_start or process_exit delta.
type ProcessChange struct {
	Type    string
	Payload map[string]any
}

// ProcessMonitor tracks process state for poll reconciliation and event dedup.
type ProcessMonitor struct {
	Logger *log.Logger
	mu     sync.Mutex
	seen   map[int]processRow
	ready  bool
}

func NewProcessMonitor(logger *log.Logger) *ProcessMonitor {
	return &ProcessMonitor{
		Logger: logger,
		seen:   map[int]processRow{},
	}
}

// Observe records an event-driven change and reports whether it should be posted.
func (m *ProcessMonitor) Observe(c ProcessChange) bool {
	pid, ok := pidFromPayload(c.Payload)
	if !ok {
		return true
	}
	row := rowFromPayload(c.Payload)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = true

	switch c.Type {
	case constants.TypeProcessStart:
		if _, exists := m.seen[pid]; exists {
			return false
		}
		if row.PID == 0 {
			row.PID = pid
		}
		enrichParentComm(c.Payload, row.PPID, m.seen)
		m.seen[pid] = row
		return true
	case constants.TypeProcessExit:
		if row.PID == 0 {
			row.PID = pid
		}
		if prev, exists := m.seen[pid]; exists {
			row = mergeExitRow(row, prev)
		}
		delete(m.seen, pid)
		// Payload is a map reference; write back so the caller enqueues enriched fields.
		c.Payload["pid"] = row.PID
		c.Payload["ppid"] = row.PPID
		c.Payload["user"] = row.User
		c.Payload["comm"] = row.Comm
		c.Payload["executable"] = row.Executable
		c.Payload["cmdline"] = row.Cmdline
		return true
	default:
		return true
	}
}

func (m *ProcessMonitor) Poll() []ProcessChange {
	rows, err := listProcesses()
	if err != nil {
		m.logf("process poll: %v", err)
		return nil
	}
	current := make(map[int]processRow, len(rows))
	for _, row := range rows {
		current[row.PID] = row
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.ready {
		m.seen = current
		m.ready = true
		return nil
	}

	var changes []ProcessChange
	var capped bool

	for pid, row := range current {
		if _, ok := m.seen[pid]; ok {
			continue
		}
		changes = append(changes, ProcessChange{
			Type:    constants.TypeProcessStart,
			Payload: processPayloadWithParent(row, current),
		})
		if len(changes) >= maxProcessEventsPerPoll {
			m.logf("process poll: capped at %d starts", maxProcessEventsPerPoll)
			capped = true
			break
		}
	}
	for pid, row := range m.seen {
		if _, ok := current[pid]; ok {
			continue
		}
		changes = append(changes, ProcessChange{
			Type:    constants.TypeProcessExit,
			Payload: processPayload(row),
		})
	}

	applyPollChanges(m.seen, current, changes, capped)
	return changes
}

func applyPollChanges(seen map[int]processRow, current map[int]processRow, changes []ProcessChange, capped bool) {
	if !capped {
		for k := range seen {
			delete(seen, k)
		}
		for pid, row := range current {
			seen[pid] = row
		}
		return
	}

	for _, ch := range changes {
		pid, ok := pidFromPayload(ch.Payload)
		if !ok {
			continue
		}
		switch ch.Type {
		case constants.TypeProcessStart:
			seen[pid] = rowFromPayload(ch.Payload)
		case constants.TypeProcessExit:
			delete(seen, pid)
		}
	}
}

func processPayload(row processRow) map[string]any {
	return map[string]any{
		"pid":        row.PID,
		"ppid":       row.PPID,
		"user":       row.User,
		"comm":       row.Comm,
		"executable": row.Executable,
		"cmdline":    row.Cmdline,
	}
}

func processPayloadWithParent(row processRow, byPID map[int]processRow) map[string]any {
	payload := processPayload(row)
	enrichParentComm(payload, row.PPID, byPID)
	return payload
}

func enrichParentComm(payload map[string]any, ppid int, byPID map[int]processRow) {
	if ppid <= 0 || payload == nil {
		return
	}
	if existing := stringFromAny(payload["parent_comm"]); existing != "" {
		return
	}
	parent, ok := byPID[ppid]
	if !ok || parent.Comm == "" {
		return
	}
	payload["parent_comm"] = parent.Comm
}

func rowFromPayload(p map[string]any) processRow {
	return processRow{
		PID:        intFromAny(p["pid"]),
		PPID:       intFromAny(p["ppid"]),
		User:       stringFromAny(p["user"]),
		Comm:       stringFromAny(p["comm"]),
		Executable: stringFromAny(p["executable"]),
		Cmdline:    stringFromAny(p["cmdline"]),
	}
}

func pidFromPayload(p map[string]any) (int, bool) {
	pid := intFromAny(p["pid"])
	if pid <= 0 {
		return 0, false
	}
	return pid, true
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func mergeExitRow(event, seen processRow) processRow {
	if event.Comm == "" {
		event.Comm = seen.Comm
	}
	if event.Executable == "" {
		event.Executable = seen.Executable
	}
	if event.Cmdline == "" {
		event.Cmdline = seen.Cmdline
	}
	if event.PPID == 0 {
		event.PPID = seen.PPID
	}
	if event.User == "" {
		event.User = seen.User
	}
	return event
}

func (m *ProcessMonitor) logf(format string, args ...any) {
	if m.Logger != nil {
		m.Logger.Printf(format, args...)
	}
}
