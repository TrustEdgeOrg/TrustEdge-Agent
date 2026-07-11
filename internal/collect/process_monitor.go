package collect

import (
	"log"

	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
)

type processRow struct {
	PID        int
	PPID       int
	User       string
	Comm       string
	Executable string
}

const maxProcessEventsPerPoll = 100

// ProcessChange is a process_start or process_exit delta.
type ProcessChange struct {
	Type    string
	Payload map[string]any
}

// ProcessMonitor diffs process tables and emits start/exit changes.
type ProcessMonitor struct {
	Logger *log.Logger
	seen   map[int]processRow
	ready  bool
}

func NewProcessMonitor(logger *log.Logger) *ProcessMonitor {
	return &ProcessMonitor{
		Logger: logger,
		seen:   map[int]processRow{},
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

	if !m.ready {
		m.seen = current
		m.ready = true
		return nil
	}

	var changes []ProcessChange
	for pid, row := range current {
		if _, ok := m.seen[pid]; ok {
			continue
		}
		changes = append(changes, ProcessChange{
			Type:    constants.TypeProcessStart,
			Payload: processPayload(row),
		})
		if len(changes) >= maxProcessEventsPerPoll {
			m.logf("process poll: capped at %d starts", maxProcessEventsPerPoll)
			break
		}
	}
	for pid, row := range m.seen {
		if _, ok := current[pid]; ok {
			continue
		}
		changes = append(changes, ProcessChange{
			Type: constants.TypeProcessExit,
			Payload: map[string]any{
				"pid":        row.PID,
				"ppid":       row.PPID,
				"user":       row.User,
				"comm":       row.Comm,
				"executable": row.Executable,
			},
		})
	}
	m.seen = current
	return changes
}

func processPayload(row processRow) map[string]any {
	return map[string]any{
		"pid":        row.PID,
		"ppid":       row.PPID,
		"user":       row.User,
		"comm":       row.Comm,
		"executable": row.Executable,
	}
}

func (m *ProcessMonitor) logf(format string, args ...any) {
	if m.Logger != nil {
		m.Logger.Printf(format, args...)
	}
}
