package agent

import (
	"context"
	"log"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// DefaultShutdownFlushTimeout bounds the final upload attempt after cancel.
const DefaultShutdownFlushTimeout = 3 * time.Second

// BatcherOptions configure flush sizing and the durable offline queue.
type BatcherOptions struct {
	MaxSize              int
	FlushEvery           time.Duration
	QueuePath            string
	Capacity             int
	MaxBackoff           time.Duration
	ShutdownFlushTimeout time.Duration
}

// EventBatcher collects telemetry into a durable ring and flushes batches to the API.
type EventBatcher struct {
	clock                clock.Clock
	deviceID             func() string
	postBatch            func(context.Context, []models.Event) error
	log                  *log.Logger
	maxSize              int
	flushEvery           time.Duration
	maxBackoff           time.Duration
	shutdownFlushTimeout time.Duration
	queue                *EventRing
	wake                 chan struct{}
}

func NewEventBatcher(clk clock.Clock, deviceID func() string, postBatch func(context.Context, []models.Event) error, logger *log.Logger, opts BatcherOptions) (*EventBatcher, error) {
	if opts.MaxSize <= 0 {
		opts.MaxSize = 32
	}
	if opts.FlushEvery <= 0 {
		opts.FlushEvery = 2 * time.Second
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 60 * time.Second
	}
	if opts.ShutdownFlushTimeout <= 0 {
		opts.ShutdownFlushTimeout = DefaultShutdownFlushTimeout
	}
	ring, err := OpenEventRing(opts.QueuePath, opts.Capacity)
	if err != nil {
		return nil, err
	}
	b := &EventBatcher{
		clock:                clk,
		deviceID:             deviceID,
		postBatch:            postBatch,
		log:                  logger,
		maxSize:              opts.MaxSize,
		flushEvery:           opts.FlushEvery,
		maxBackoff:           opts.MaxBackoff,
		shutdownFlushTimeout: opts.ShutdownFlushTimeout,
		queue:                ring,
		wake:                 make(chan struct{}, 1),
	}
	if pending := ring.Len(); pending > 0 && logger != nil {
		logger.Printf("event queue: restored %d pending event(s)", pending)
	}
	return b, nil
}

func (b *EventBatcher) Enqueue(typ string, payload map[string]any) {
	ev := models.NewEvent(b.clock, b.deviceID(), typ, payload)
	dropped, err := b.queue.Push(ev)
	if err != nil && b.log != nil {
		b.log.Printf("event queue persist: %v", err)
	}
	if dropped && b.log != nil {
		b.log.Printf("event queue full: dropped oldest (pending=%d dropped_total=%d)", b.queue.Len(), b.queue.Dropped())
	}
	if b.queue.Len() >= b.maxSize {
		b.signalFlush()
	}
}

func (b *EventBatcher) Run(ctx context.Context) {
	backoff := b.flushEvery
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Parent ctx is cancelled; use a short independent deadline for a
			// best-effort final flush. Unacked events remain in the durable ring.
			flushCtx, cancel := context.WithTimeout(context.Background(), b.shutdownFlushTimeout)
			_ = b.Flush(flushCtx)
			cancel()
			return
		case <-timer.C:
			err := b.Flush(ctx)
			if err != nil {
				backoff = nextBackoff(backoff, b.flushEvery, b.maxBackoff)
			} else {
				backoff = b.flushEvery
				if b.queue.Len() > 0 {
					b.signalFlush()
				}
			}
			timer.Reset(backoff)
		case <-b.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			err := b.Flush(ctx)
			if err != nil {
				backoff = nextBackoff(backoff, b.flushEvery, b.maxBackoff)
			} else {
				backoff = b.flushEvery
				if b.queue.Len() > 0 {
					b.signalFlush()
				}
			}
			timer.Reset(backoff)
		}
	}
}

// Flush peeks a batch, posts it, and only acks on success so failures stay queued.
func (b *EventBatcher) Flush(ctx context.Context) error {
	batch := b.queue.Peek(b.maxSize)
	if len(batch) == 0 {
		return nil
	}
	if err := b.postBatch(ctx, batch); err != nil {
		if b.log != nil {
			b.log.Printf("post batch (%d events, pending=%d): %v", len(batch), b.queue.Len(), err)
		}
		return err
	}
	if err := b.queue.Ack(len(batch)); err != nil {
		if b.log != nil {
			b.log.Printf("event queue ack: %v", err)
		}
		return err
	}
	if b.log != nil {
		b.log.Printf("posted batch (%d events, pending=%d)", len(batch), b.queue.Len())
	}
	return nil
}

func (b *EventBatcher) signalFlush() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

func nextBackoff(current, min, max time.Duration) time.Duration {
	next := current * 2
	if next < min {
		next = min
	}
	if next > max {
		return max
	}
	return next
}
