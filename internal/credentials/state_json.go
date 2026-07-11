package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type persistedState struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token,omitempty"`
}

func loadStateFile(path string) (persistedState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return persistedState{DeviceID: newDeviceID()}, nil
		}
		return persistedState{}, err
	}
	var st persistedState
	if err := json.Unmarshal(data, &st); err != nil {
		return persistedState{}, err
	}
	if st.DeviceID == "" {
		st.DeviceID = newDeviceID()
	}
	return st, nil
}

func saveStateFile(path string, deviceID string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	st := persistedState{DeviceID: deviceID}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func saveStateFileWithToken(path string, deviceID, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	st := persistedState{DeviceID: deviceID, DeviceToken: token}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func newDeviceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "dev_" + hex.EncodeToString(b)
}
