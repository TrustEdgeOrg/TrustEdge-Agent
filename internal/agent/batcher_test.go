package agent

import (
	"context"
	"sync"
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
	b := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		mu.Lock()
		batch = append(batch, ev...)
		mu.Unlock()
		return nil
	}, nil, 3, time.Hour)

	b.Enqueue("process_start", map[string]any{"pid": 1})
	b.Enqueue("process_start", map[string]any{"pid": 2})
	if len(batch) != 0 {
		t.Fatalf("batch early=%d", len(batch))
	}
	b.Enqueue("process_start", map[string]any{"pid": 3})
	b.Flush()

	mu.Lock()
	defer mu.Unlock()
	if len(batch) != 3 {
		t.Fatalf("batch=%d want 3", len(batch))
	}
}

func TestEventBatcherFlushOnInterval(t *testing.T) {
	var posted int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewEventBatcher(clock.Real{}, func() string { return "dev_test" }, func(ev []models.Event) error {
		posted += len(ev)
		return nil
	}, nil, 100, 50*time.Millisecond)

	go b.Run(ctx)
	b.Enqueue("client_details", map[string]any{"hostname": "test"})
	time.Sleep(120 * time.Millisecond)
	cancel()
	b.Flush()

	if posted != 1 {
		t.Fatalf("posted=%d want 1", posted)
	}
}
