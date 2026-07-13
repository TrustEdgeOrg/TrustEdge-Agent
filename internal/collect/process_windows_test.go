//go:build windows

package collect

import "testing"

func TestParseWinProcessJSON(t *testing.T) {
	sample := `[{"ProcessId":123,"ParentProcessId":1,"Name":"curl.exe","ExecutablePath":"C:\\curl.exe"}]`
	rows, err := parseWinProcessJSON(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Comm != "curl.exe" {
		t.Fatalf("rows=%+v", rows)
	}

	one := `{"ProcessId":9,"ParentProcessId":4,"Name":"System","ExecutablePath":""}`
	rows, err = parseWinProcessJSON(one)
	if err != nil || len(rows) != 1 || rows[0].PID != 9 {
		t.Fatalf("one=%+v err=%v", rows, err)
	}
}
