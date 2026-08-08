package apps

import (
	"path/filepath"
	"testing"
)

func TestEnclosingAppPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/Applications/Cursor.app/Contents/MacOS/Cursor", "/Applications/Cursor.app"},
		{"/Applications/Cursor.app/Contents/Frameworks/Cursor Helper.app/Contents/MacOS/Cursor Helper", "/Applications/Cursor.app"},
		{"/usr/bin/python3", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := EnclosingAppPath(tt.in)
		if got != tt.want {
			t.Fatalf("EnclosingAppPath(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
	// Nested helper: outermost .app is Cursor.app
	nested := filepath.Join("/Applications", "Cursor.app", "Contents", "Frameworks", "Cursor Helper (GPU).app", "Contents", "MacOS", "Cursor Helper (GPU)")
	if got := EnclosingAppPath(nested); got != "/Applications/Cursor.app" {
		t.Fatalf("nested=%q", got)
	}
}
