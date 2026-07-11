//go:build darwin

package credentials

import (
	"errors"
	"log"

	"github.com/zalando/go-keyring"
)

type darwinStore struct {
	statePath string
	log       *log.Logger
}

func newPlatformStore(statePath string, logger *log.Logger) Store {
	return newDarwinStore(statePath, logger)
}

func newDarwinStore(statePath string, logger *log.Logger) Store {
	if logger == nil {
		logger = log.Default()
	}
	return &darwinStore{statePath: statePath, log: logger}
}

func (s *darwinStore) Load() (string, string, error) {
	st, err := loadStateFile(s.statePath)
	if err != nil {
		return "", "", err
	}
	token, err := keyring.Get(keyringService, keyringAccount)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", "", err
	}
	if token == "" && st.DeviceToken != "" {
		if err := keyring.Set(keyringService, keyringAccount, st.DeviceToken); err != nil {
			return "", "", err
		}
		token = st.DeviceToken
		if err := saveStateFile(s.statePath, st.DeviceID); err != nil {
			return "", "", err
		}
		s.log.Printf("migrated device token from state.json to Keychain")
	}
	return st.DeviceID, token, nil
}

func (s *darwinStore) Save(deviceID, token string) error {
	if err := saveStateFile(s.statePath, deviceID); err != nil {
		return err
	}
	if token == "" {
		return keyring.Delete(keyringService, keyringAccount)
	}
	return keyring.Set(keyringService, keyringAccount, token)
}

func (s *darwinStore) ClearToken() error {
	if err := keyring.Delete(keyringService, keyringAccount); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}
