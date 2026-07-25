//go:build darwin

package collect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDarwinSecurityWatcherEmitsOnPlistWrite(t *testing.T) {
	dir := t.TempDir()
	agents := filepath.Join(dir, "Library", "LaunchAgents")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		t.Fatal(err)
	}

	w := &darwinSecurityWatcher{
		debounce: 30 * time.Millisecond,
		dirs:     []string{agents},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := w.Run(ctx)

	// Allow kqueue registration to settle.
	time.Sleep(50 * time.Millisecond)

	plist := filepath.Join(agents, "com.trustedge.test.plist")
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>Label</key><string>com.trustedge.test</string></dict></plist>`
	if err := os.WriteFile(plist, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-out:
	case <-time.After(2 * time.Second):
		t.Fatal("expected wake after plist write")
	}
}
