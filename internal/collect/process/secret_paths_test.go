package process

import "testing"

func TestPathLooksSecret(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/Users/me/proj/.env", true},
		{"/Users/me/proj/.env.local", true},
		{"/Users/me/.ssh/id_rsa", true},
		{"/Users/me/.aws/credentials", true},
		{"/tmp/secrets.json", true},
		{"/Users/me/proj/README.md", false},
		{"/usr/bin/env", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := pathLooksSecret(tc.path); got != tc.want {
			t.Fatalf("pathLooksSecret(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}

func TestFileOpenPayload(t *testing.T) {
	row := processRow{PID: 42, PPID: 1, Comm: "Cursor", Executable: "/Applications/Cursor.app/Contents/MacOS/Cursor"}
	p := fileOpenPayload(row, "/Users/me/proj/.env")
	if p["path"] != "/Users/me/proj/.env" {
		t.Fatalf("path=%v", p["path"])
	}
	if p["operation"] != "open" {
		t.Fatalf("operation=%v", p["operation"])
	}
	if p["pid"] != 42 {
		t.Fatalf("pid=%v", p["pid"])
	}
}
