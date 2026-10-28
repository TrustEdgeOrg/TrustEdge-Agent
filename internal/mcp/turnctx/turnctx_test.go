package turnctx

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ts := time.Now().UTC()
	if err := Write(dir, "conv-1", "gen-1", ts); err != nil {
		t.Fatal(err)
	}
	got, ok := Read(dir, time.Hour)
	if !ok {
		t.Fatal("expected active turn")
	}
	if got.SessionID != "conv-1" || got.TurnID != "gen-1" {
		t.Fatalf("got=%+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, activeTurnFile)); err != nil {
		t.Fatal(err)
	}
}

func TestReadRejectsStale(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().UTC().Add(-2 * time.Hour)
	if err := Write(dir, "conv-1", "gen-1", old); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read(dir, 30*time.Minute); ok {
		t.Fatal("expected stale turn to be ignored")
	}
}
