package apps

import "testing"

func TestEnclosingAppPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/Applications/Cursor.app/Contents/MacOS/Cursor", "/Applications/Cursor.app"},
		{"/Applications/Cursor.app/Contents/Frameworks/Cursor Helper.app/Contents/MacOS/Cursor Helper", "/Applications/Cursor.app"},
		{`\Applications\Cursor.app\Contents\MacOS\Cursor`, "/Applications/Cursor.app"},
		{"/usr/bin/python3", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := EnclosingAppPath(tt.in)
		if got != tt.want {
			t.Fatalf("EnclosingAppPath(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
	nested := "/Applications/Cursor.app/Contents/Frameworks/Cursor Helper (GPU).app/Contents/MacOS/Cursor Helper (GPU)"
	if got := EnclosingAppPath(nested); got != "/Applications/Cursor.app" {
		t.Fatalf("nested=%q", got)
	}
}

func TestPathKeyStableAcrossSeparators(t *testing.T) {
	a := pathKey(`/Applications/Cursor.app`)
	b := pathKey(`\Applications\Cursor.app`)
	if a != b {
		t.Fatalf("pathKey mismatch %q vs %q", a, b)
	}
}
