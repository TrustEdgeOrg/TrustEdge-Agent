//go:build darwin || linux || windows

package credentials

import (
	"errors"
	"log"

	"github.com/zalando/go-keyring"
)

type keyringStore struct {
	statePath string
	log       *log.Logger
}

func newPlatformStore(statePath string, logger *log.Logger) Store {
	if logger == nil {
		logger = log.Default()
	}
	return &keyringStore{statePath: statePath, log: logger}
}

func keyringGetToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringAccount)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", err
	}
	if token != "" {
		return token, nil
	}
	legacy, legacyErr := keyring.Get(keyringServiceLegacy, keyringAccount)
	if legacyErr != nil && !errors.Is(legacyErr, keyring.ErrNotFound) {
		return "", legacyErr
	}
	if legacy == "" {
		return "", nil
	}
	if err := keyring.Set(keyringService, keyringAccount, legacy); err != nil {
		return legacy, nil
	}
	_ = keyring.Delete(keyringServiceLegacy, keyringAccount)
	return legacy, nil
}

func (s *keyringStore) Load() (string, string, error) {
	st, err := loadStateFile(s.statePath)
	if err != nil {
		return "", "", err
	}
	token, err := keyringGetToken()
	if err != nil {
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
		s.log.Printf("migrated device token from state.json to OS credential store")
	}
	return st.DeviceID, token, nil
}

func (s *keyringStore) Save(deviceID, token string) error {
	if err := saveStateFile(s.statePath, deviceID); err != nil {
		return err
	}
	if token == "" {
		return keyring.Delete(keyringService, keyringAccount)
	}
	return keyring.Set(keyringService, keyringAccount, token)
}

func (s *keyringStore) ClearToken() error {
	if err := keyring.Delete(keyringService, keyringAccount); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	_ = keyring.Delete(keyringServiceLegacy, keyringAccount)
	return nil
}
