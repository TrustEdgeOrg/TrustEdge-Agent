//go:build !darwin && !linux && !windows

package credentials

import (
	"log"
	"sync"
)

type fileStore struct {
	statePath string
	log       *log.Logger
	warnOnce  sync.Once
}

func newPlatformStore(statePath string, logger *log.Logger) Store {
	return newFileStore(statePath, logger)
}

func newFileStore(statePath string, logger *log.Logger) Store {
	if logger == nil {
		logger = log.Default()
	}
	return &fileStore{statePath: statePath, log: logger}
}

func (s *fileStore) warnInsecure() {
	s.warnOnce.Do(func() {
		s.log.Printf("warning: device token stored in plaintext state file; use macOS, Linux, or Windows for credential-store-backed tokens")
	})
}

func (s *fileStore) Load() (string, string, error) {
	s.warnInsecure()
	st, err := loadStateFile(s.statePath)
	if err != nil {
		return "", "", err
	}
	return st.DeviceID, st.DeviceToken, nil
}

func (s *fileStore) Save(deviceID, token string) error {
	s.warnInsecure()
	return saveStateFileWithToken(s.statePath, deviceID, token)
}

func (s *fileStore) ClearToken() error {
	s.warnInsecure()
	st, err := loadStateFile(s.statePath)
	if err != nil {
		return err
	}
	return saveStateFileWithToken(s.statePath, st.DeviceID, "")
}
