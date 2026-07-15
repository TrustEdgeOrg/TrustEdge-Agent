package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// EventRing is a bounded FIFO of telemetry events. When full, the oldest
// events are overwritten. With a non-empty path the ring is persisted
// atomically so pending events survive process restarts.
type EventRing struct {
	path     string
	capacity int

	mu      sync.Mutex
	events  []models.Event
	dropped uint64
}

type ringFile struct {
	Capacity int            `json:"capacity"`
	Dropped  uint64         `json:"dropped"`
	Events   []models.Event `json:"events"`
}

// OpenEventRing loads an existing queue from path, or starts empty.
// capacity <= 0 defaults to constants.DefaultEventQueueCapacity. An empty
// path disables durability (in-memory only) but still applies the ring
// bound and overwrite policy.
func OpenEventRing(path string, capacity int) (*EventRing, error) {
	if capacity <= 0 {
		capacity = constants.DefaultEventQueueCapacity
	}
	r := &EventRing{
		path:     path,
		capacity: capacity,
		events:   make([]models.Event, 0, min(capacity, 64)),
	}
	if path == "" {
		return r, nil
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

// Push appends an event. If the ring is full the oldest event is dropped.
// Returns true when an older event was overwritten.
func (r *EventRing) Push(ev models.Event) (droppedOldest bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.events) >= r.capacity {
		copy(r.events[0:], r.events[1:])
		r.events[r.capacity-1] = ev
		r.events = r.events[:r.capacity]
		r.dropped++
		droppedOldest = true
	} else {
		r.events = append(r.events, ev)
	}
	return droppedOldest, r.persistLocked()
}

// Len returns the number of pending events.
func (r *EventRing) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// Dropped returns how many oldest events were overwritten since open
// (including counts restored from disk).
func (r *EventRing) Dropped() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dropped
}

// Peek returns up to n events from the front without removing them.
func (r *EventRing) Peek(n int) []models.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || len(r.events) == 0 {
		return nil
	}
	if n > len(r.events) {
		n = len(r.events)
	}
	out := make([]models.Event, n)
	copy(out, r.events[:n])
	return out
}

// Ack removes the first n events after a successful upload.
func (r *EventRing) Ack(n int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 {
		return nil
	}
	if n > len(r.events) {
		n = len(r.events)
	}
	r.events = append([]models.Event(nil), r.events[n:]...)
	return r.persistLocked()
}

func (r *EventRing) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f ringFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	r.dropped = f.Dropped
	if len(f.Events) > r.capacity {
		overflow := len(f.Events) - r.capacity
		r.events = append([]models.Event(nil), f.Events[overflow:]...)
		r.dropped += uint64(overflow)
	} else {
		r.events = append([]models.Event(nil), f.Events...)
	}
	return nil
}

func (r *EventRing) persistLocked() error {
	if r.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	f := ringFile{
		Capacity: r.capacity,
		Dropped:  r.dropped,
		Events:   r.events,
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}
