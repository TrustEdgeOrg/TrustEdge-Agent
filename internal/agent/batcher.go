package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

// EventBatcher collects telemetry and flushes batches to the API.
type EventBatcher struct {
	clock      clock.Clock
	deviceID   func() string
	postBatch  func([]models.Event) error
	log        *log.Logger
	maxSize    int
	flushEvery time.Duration

	mu   sync.Mutex
	buf  []models.Event
	wake chan struct{}
}

func NewEventBatcher(clk clock.Clock, deviceID func() string, postBatch func([]models.Event) error, logger *log.Logger, maxSize int, flushEvery time.Duration) *EventBatcher {
	if maxSize <= 0 {
		maxSize = 32
	}
	if flushEvery <= 0 {
		flushEvery = 2 * time.Second
	}
	return &EventBatcher{
		clock:      clk,
		deviceID:   deviceID,
		postBatch:  postBatch,
		log:        logger,
		maxSize:    maxSize,
		flushEvery: flushEvery,
		wake:       make(chan struct{}, 1),
	}
}

func (b *EventBatcher) Enqueue(typ string, payload map[string]any) {
	b.mu.Lock()
	b.buf = append(b.buf, models.NewEvent(b.clock, b.deviceID(), typ, payload))
	full := len(b.buf) >= b.maxSize
	b.mu.Unlock()
	if full {
		b.signalFlush()
	}
}

func (b *EventBatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(b.flushEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.Flush()
			return
		case <-ticker.C:
			b.Flush()
		case <-b.wake:
			b.Flush()
		}
	}
}

func (b *EventBatcher) Flush() {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := make([]models.Event, len(b.buf))
	copy(batch, b.buf)
	b.buf = b.buf[:0]
	b.mu.Unlock()

	if err := b.postBatch(batch); err != nil {
		if b.log != nil {
			b.log.Printf("post batch (%d events): %v", len(batch), err)
		}
		return
	}
	if b.log != nil {
		b.log.Printf("posted batch (%d events)", len(batch))
	}
}

func (b *EventBatcher) signalFlush() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}
