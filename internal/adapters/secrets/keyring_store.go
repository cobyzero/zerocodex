package secrets

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	serviceName    = "com.cobyzero.zerocodex"
	deepSeekUserID = "deepseek_api_key"
)

type KeyringStore struct{}

func NewKeyringStore() *KeyringStore {
	return &KeyringStore{}
}

func (s *KeyringStore) SaveDeepSeekAPIKey(apiKey string) error {
	return keyring.Set(serviceName, deepSeekUserID, apiKey)
}

func (s *KeyringStore) LoadDeepSeekAPIKey() (string, error) {
	secret, err := keyring.Get(serviceName, deepSeekUserID)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return secret, err
}
