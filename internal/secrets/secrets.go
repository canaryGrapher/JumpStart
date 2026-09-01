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
	// KeyGitHubToken holds the bare GitHub access token. It stays the
	// canonical place any caller can read a usable token from, including
	// helpers outside the sync engine (git push, release publishing).
	KeyGitHubToken = "github_token"

	// KeyGitHubTokenSet holds the JSON github.TokenSet behind that token:
	// the refresh token and both expiry instants. GitHub Apps issue user
	// tokens that expire in hours, so without this the app can only ask
	// the user to sign in again. Written alongside KeyGitHubToken, which
	// is always kept in step with the set's current access token.
	KeyGitHubTokenSet = "github_token_set"

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
