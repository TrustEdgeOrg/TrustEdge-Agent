package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type AgentState struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token"`
}

func Load(path string) (*AgentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AgentState{DeviceID: newDeviceID()}, nil
		}
		return nil, err
	}
	var st AgentState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.DeviceID == "" {
		st.DeviceID = newDeviceID()
	}
	return &st, nil
}

func (s *AgentState) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
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

func NewToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "tok_" + hex.EncodeToString(b)
}
