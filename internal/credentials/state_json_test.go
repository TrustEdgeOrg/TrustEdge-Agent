package credentials

import (
	"path/filepath"
	"testing"
)

func TestStateFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := saveStateFile(path, "dev_test"); err != nil {
		t.Fatal(err)
	}
	st, err := loadStateFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.DeviceID != "dev_test" {
		t.Fatalf("device_id=%q", st.DeviceID)
	}
	if st.DeviceToken != "" {
		t.Fatalf("expected empty legacy token field, got %q", st.DeviceToken)
	}
}

func TestLegacyTokenInStateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := saveStateFileWithToken(path, "dev_legacy", "tok_legacy"); err != nil {
		t.Fatal(err)
	}
	st, err := loadStateFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.DeviceToken != "tok_legacy" {
		t.Fatalf("token=%q", st.DeviceToken)
	}
}
