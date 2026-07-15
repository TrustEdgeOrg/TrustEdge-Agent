package agent

import (
	"sync/atomic"
	"time"
)

// Metrics tracks lightweight agent delivery counters for structured status logs.
type Metrics struct {
	UploadSuccess   atomic.Uint64
	UploadFail      atomic.Uint64
	AuthRecover     atomic.Uint64
	QueueDropped    atomic.Uint64
	lastUploadUnixN atomic.Int64
}

func (m *Metrics) RecordUploadSuccess() {
	if m == nil {
		return
	}
	m.UploadSuccess.Add(1)
	m.lastUploadUnixN.Store(time.Now().UnixNano())
}

func (m *Metrics) RecordUploadFail() {
	if m == nil {
		return
	}
	m.UploadFail.Add(1)
}

func (m *Metrics) RecordAuthRecover() {
	if m == nil {
		return
	}
	m.AuthRecover.Add(1)
}

func (m *Metrics) RecordQueueDropped() {
	if m == nil {
		return
	}
	m.QueueDropped.Add(1)
}

func (m *Metrics) LastUploadAge(now time.Time) (ageSec float64, ok bool) {
	if m == nil {
		return 0, false
	}
	ns := m.lastUploadUnixN.Load()
	if ns == 0 {
		return 0, false
	}
	return now.Sub(time.Unix(0, ns)).Seconds(), true
}
