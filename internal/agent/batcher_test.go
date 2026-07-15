package agent

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func TestEventBatcherFlushOnSize(t *testing.T) {
	var (
		mu    sync.Mutex
		batch []models.Event
	)
	b, err := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		mu.Lock()
		batch = append(batch, ev...)
		mu.Unlock()
		return nil
	}, nil, BatcherOptions{MaxSize: 3, FlushEvery: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	b.Enqueue("process_start", map[string]any{"pid": 1})
	b.Enqueue("process_start", map[string]any{"pid": 2})
	if len(batch) != 0 {
		t.Fatalf("batch early=%d", len(batch))
	}
	b.Enqueue("process_start", map[string]any{"pid": 3})
	if err := b.Flush(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(batch) != 3 {
		t.Fatalf("batch=%d want 3", len(batch))
	}
	if b.queue.Len() != 0 {
		t.Fatalf("pending=%d", b.queue.Len())
	}
}

func TestEventBatcherFlushOnInterval(t *testing.T) {
	var posted int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, err := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		posted += len(ev)
		return nil
	}, nil, BatcherOptions{MaxSize: 100, FlushEvery: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	go b.Run(ctx)
	b.Enqueue("client_details", map[string]any{"hostname": "test"})
	time.Sleep(120 * time.Millisecond)
	cancel()
	time.Sleep(30 * time.Millisecond)

	if posted != 1 {
		t.Fatalf("posted=%d want 1", posted)
	}
}

func TestEventBatcherRetainsOnFailure(t *testing.T) {
	var calls atomic.Int32
	b, err := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		calls.Add(1)
		return errors.New("api down")
	}, nil, BatcherOptions{MaxSize: 10, FlushEvery: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	b.Enqueue("network_summary", map[string]any{"public_ip": "1.2.3.4"})
	if err := b.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
	if b.queue.Len() != 1 {
		t.Fatalf("pending=%d want 1", b.queue.Len())
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestEventBatcherRetriesAfterFailure(t *testing.T) {
	var (
		mu     sync.Mutex
		posted []models.Event
	)
	var failFirst atomic.Bool
	failFirst.Store(true)

	dir := t.TempDir()
	b, err := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		if failFirst.Load() {
			failFirst.Store(false)
			return errors.New("temporary")
		}
		mu.Lock()
		posted = append(posted, ev...)
		mu.Unlock()
		return nil
	}, nil, BatcherOptions{
		MaxSize:    10,
		FlushEvery: time.Hour,
		QueuePath:  filepath.Join(dir, "events.queue.json"),
		Capacity:   64,
	})
	if err != nil {
		t.Fatal(err)
	}

	b.Enqueue("action_summary", map[string]any{"presence": "active"})
	if err := b.Flush(); err == nil {
		t.Fatal("expected first flush to fail")
	}
	if err := b.Flush(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(posted) != 1 {
		t.Fatalf("posted=%d", len(posted))
	}
	if b.queue.Len() != 0 {
		t.Fatalf("pending=%d", b.queue.Len())
	}

	// Survive reopen with empty queue after successful ack.
	r, err := OpenEventRing(filepath.Join(dir, "events.queue.json"), 64)
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != 0 {
		t.Fatalf("disk pending=%d", r.Len())
	}
}
