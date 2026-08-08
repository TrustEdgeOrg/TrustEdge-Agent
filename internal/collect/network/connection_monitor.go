package network

import (
	"log"
	"sync"
)

const maxConnectionEventsPerPoll = 40

// ConnectionMonitor emits newly observed established TCP sockets.
type ConnectionMonitor struct {
	Logger *log.Logger
	// List is injectable for tests; defaults to listEstablishedConnections.
	List func() ([]EstablishedConn, error)

	mu    sync.Mutex
	seen  map[string]EstablishedConn
	ready bool
}

func NewConnectionMonitor(logger *log.Logger) *ConnectionMonitor {
	return &ConnectionMonitor{
		Logger: logger,
		List:   listEstablishedConnections,
		seen:   map[string]EstablishedConn{},
	}
}

// Poll returns newly seen connections since the previous poll.
// The first poll seeds a silent baseline and returns nil.
func (m *ConnectionMonitor) Poll() []EstablishedConn {
	listFn := m.List
	if listFn == nil {
		listFn = listEstablishedConnections
	}
	conns, err := listFn()
	if err != nil {
		m.logf("connection poll: %v", err)
		return nil
	}

	current := make(map[string]EstablishedConn, len(conns))
	for _, c := range conns {
		if c.PID <= 0 || c.RemoteAddr == "" || c.RemotePort <= 0 {
			continue
		}
		fp := c.Fingerprint()
		current[fp] = c
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.ready {
		m.seen = current
		m.ready = true
		return nil
	}

	var added []EstablishedConn
	for fp, c := range current {
		if _, ok := m.seen[fp]; ok {
			continue
		}
		added = append(added, c.WithHostname())
		if len(added) >= maxConnectionEventsPerPoll {
			m.logf("connection poll: capped at %d new sockets", maxConnectionEventsPerPoll)
			break
		}
	}
	m.seen = current
	return added
}

func (m *ConnectionMonitor) logf(format string, args ...any) {
	if m.Logger != nil {
		m.Logger.Printf(format, args...)
	}
}
