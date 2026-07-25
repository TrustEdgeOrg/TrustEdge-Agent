package collect

import (
	"context"
	"testing"
	"time"
)

func TestDebounceWakeCoalescesBursts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan struct{}, 8)
	out := debounceWake(ctx, in, 50*time.Millisecond)

	for i := 0; i < 5; i++ {
		in <- struct{}{}
	}

	select {
	case <-out:
		t.Fatal("wake arrived before debounce elapsed")
	case <-time.After(20 * time.Millisecond):
	}

	select {
	case <-out:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected coalesced wake")
	}

	select {
	case <-out:
		t.Fatal("unexpected second wake")
	case <-time.After(80 * time.Millisecond):
	}
}

func TestDebounceWakeFlushesOnInputClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan struct{}, 1)
	out := debounceWake(ctx, in, time.Hour)
	in <- struct{}{}
	close(in)

	select {
	case <-out:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected flush on input close")
	}
}
