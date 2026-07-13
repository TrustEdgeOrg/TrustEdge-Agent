package config

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultStatePath(t *testing.T) {
	path := defaultStatePath()
	if path == "" {
		t.Fatal("empty path")
	}
	switch runtime.GOOS {
	case "darwin":
		if filepath.Base(filepath.Dir(path)) != "TrustEdge Agent" {
			t.Fatalf("path=%s", path)
		}
	case "linux":
		if !strings.Contains(path, "TrustEdge Agent") {
			t.Fatalf("path=%s", path)
		}
	case "windows":
		if !strings.Contains(path, "TrustEdge Agent") {
			t.Fatalf("path=%s", path)
		}
	}
}
