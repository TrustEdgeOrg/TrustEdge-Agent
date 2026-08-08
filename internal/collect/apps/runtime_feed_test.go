package apps

import (
	"context"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestRuntimeFeedWakesOnCursorCandidate(t *testing.T) {
	feed := NewRuntimeFeed(nil, identity.NewMatcher(identity.DefaultCatalog()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go feed.Run(ctx)

	feed.ObserveChange(collect.Change{
		Type: constants.TypeProcessStart,
		Payload: map[string]any{
			"pid":        1,
			"ppid":       0,
			"comm":       "Cursor",
			"executable": "/Applications/Cursor.app/Contents/MacOS/Cursor",
		},
	})

	select {
	case <-feed.Wakes():
	case <-time.After(2 * time.Second):
		t.Fatal("expected wake for Cursor candidate")
	}
}

func TestRuntimeFeedIgnoresUnrelatedProcess(t *testing.T) {
	feed := NewRuntimeFeed(nil, identity.NewMatcher(identity.DefaultCatalog()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go feed.Run(ctx)

	feed.ObserveChange(collect.Change{
		Type: constants.TypeProcessFork,
		Payload: map[string]any{
			"pid":        2,
			"ppid":       1,
			"comm":       "Safari",
			"executable": "/Applications/Safari.app/Contents/MacOS/Safari",
		},
	})

	select {
	case <-feed.Wakes():
		t.Fatal("should not wake for Safari")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRuntimeFeedObserveIsNonBlocking(t *testing.T) {
	feed := NewRuntimeFeed(nil, nil)
	// Fill queue without running worker.
	for i := 0; i < 300; i++ {
		feed.ObserveChange(collect.Change{
			Type:    constants.TypeProcessStart,
			Payload: map[string]any{"pid": i, "comm": "Cursor", "executable": "/Applications/Cursor.app/Contents/MacOS/Cursor"},
		})
	}
}
