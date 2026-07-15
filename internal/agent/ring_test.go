package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func TestEventRingPushPeekAck(t *testing.T) {
	r, err := OpenEventRing("", 4)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		r.Push(models.Event{EventID: string(rune('a' + i - 1)), Type: "t"})
	}
	if r.Len() != 3 {
		t.Fatalf("len=%d", r.Len())
	}
	got := r.Peek(2)
	if len(got) != 2 || got[0].EventID != "a" || got[1].EventID != "b" {
		t.Fatalf("peek=%+v", got)
	}
	if err := r.Ack(2); err != nil {
		t.Fatal(err)
	}
	if r.Len() != 1 {
		t.Fatalf("len after ack=%d", r.Len())
	}
	got = r.Peek(10)
	if len(got) != 1 || got[0].EventID != "c" {
		t.Fatalf("remaining=%+v", got)
	}
}

func TestEventRingOverwriteOldest(t *testing.T) {
	r, err := OpenEventRing("", 2)
	if err != nil {
		t.Fatal(err)
	}
	r.Push(models.Event{EventID: "1"})
	r.Push(models.Event{EventID: "2"})
	dropped, err := r.Push(models.Event{EventID: "3"})
	if err != nil {
		t.Fatal(err)
	}
	if !dropped {
		t.Fatal("expected overwrite")
	}
	if r.Len() != 2 || r.Dropped() != 1 {
		t.Fatalf("len=%d dropped=%d", r.Len(), r.Dropped())
	}
	got := r.Peek(10)
	if got[0].EventID != "2" || got[1].EventID != "3" {
		t.Fatalf("got=%+v", got)
	}
}

func TestEventRingPersistsAcrossOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.queue.json")

	r1, err := OpenEventRing(path, 8)
	if err != nil {
		t.Fatal(err)
	}
	r1.Push(models.Event{
		EventID:  "evt_1",
		DeviceID: "dev",
		Type:     "client_details",
		TS:       time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
		Payload:  map[string]any{"hostname": "host"},
	})
	if err := r1.Ack(0); err != nil {
		t.Fatal(err)
	}

	r2, err := OpenEventRing(path, 8)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Len() != 1 {
		t.Fatalf("restored len=%d", r2.Len())
	}
	got := r2.Peek(1)
	if got[0].EventID != "evt_1" || got[0].Payload["hostname"] != "host" {
		t.Fatalf("restored=%+v", got[0])
	}
	if err := r2.Ack(1); err != nil {
		t.Fatal(err)
	}

	r3, err := OpenEventRing(path, 8)
	if err != nil {
		t.Fatal(err)
	}
	if r3.Len() != 0 {
		t.Fatalf("expected empty after ack, len=%d", r3.Len())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("queue file missing: %v", err)
	}
}
