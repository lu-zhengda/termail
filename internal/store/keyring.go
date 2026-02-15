package store

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const serviceName = "termail"

// KeyringTokenStore persists OAuth2 tokens in the OS keyring
// (macOS Keychain, Windows Credential Manager, or Linux Secret Service).
type KeyringTokenStore struct{}

// NewKeyringTokenStore returns a new KeyringTokenStore.
func NewKeyringTokenStore() *KeyringTokenStore {
	return &KeyringTokenStore{}
}

// SaveToken stores the given OAuth2 token in the OS keyring under the account ID.
func (k *KeyringTokenStore) SaveToken(accountID string, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	if err := keyring.Set(serviceName, accountID, string(data)); err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	return nil
}

// LoadToken retrieves the OAuth2 token for the given account ID from the OS keyring.
func (k *KeyringTokenStore) LoadToken(accountID string) (*oauth2.Token, error) {
	data, err := keyring.Get(serviceName, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load token from keyring: %w", err)
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	return &token, nil
}

// DeleteToken removes the OAuth2 token for the given account ID from the OS keyring.
func (k *KeyringTokenStore) DeleteToken(accountID string) error {
	if err := keyring.Delete(serviceName, accountID); err != nil {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}
	return nil
}
