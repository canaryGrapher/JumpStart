// Package secrets stores small secrets (git provider tokens) in the OS
// keychain via go-keyring, so tokens never touch the JSON config file.
package secrets

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

// Service is the keychain service name under which all JumpStart secrets
// are grouped.
const Service = "jumpstart"

// Common keys used for git provider tokens.
const (
	KeyGitHubToken = "github_token"
	KeyGitLabToken = "gitlab_token"
)

// SaveToken stores token under service/key in the OS keychain.
func SaveToken(service, key, token string) error {
	if err := keyring.Set(service, key, token); err != nil {
		return fmt.Errorf("saving token to keychain: %w", err)
	}
	return nil
}

// GetToken retrieves the token stored under service/key. Returns an empty
// string, nil error if no token has ever been saved.
func GetToken(service, key string) (string, error) {
	token, err := keyring.Get(service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("reading token from keychain: %w", err)
	}
	return token, nil
}

// DeleteToken removes the token stored under service/key, if any.
func DeleteToken(service, key string) error {
	if err := keyring.Delete(service, key); err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("deleting token from keychain: %w", err)
	}
	return nil
}
