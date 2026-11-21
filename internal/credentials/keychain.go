package credentials

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// EnvNoKeychain set to any value keeps secrets in files instead of the OS keychain.
const EnvNoKeychain = "RUMPTY_NO_KEYCHAIN"

const (
	keychainService = "rumpty-cli"
	keychainTimeout = 3 * time.Second
	apiKeyAccount   = "api-key"
)

var errKeychainTimeout = errors.New("keychain did not respond")

// Tests use the mock keychain explicitly instead of the real one.
var keychainEnabled = !testing.Testing()

// keychainHung is set after a keychain call times out, so the rest of the
// command doesn't wait on it again.
var keychainHung atomic.Bool

// KeychainAvailable reports whether secrets go to the OS keychain.
func KeychainAvailable() bool {
	return keychainEnabled && os.Getenv(EnvNoKeychain) == "" && !keychainHung.Load()
}

func keychainOn() bool { return KeychainAvailable() }

// StoreSecret saves secret in the OS keychain. It returns false when no
// keychain is available, so the caller can keep the secret in a file.
func StoreSecret(account, secret string) bool {
	if !keychainOn() {
		return false
	}
	return withTimeout(func() error { return keyring.Set(keychainService, account, secret) }) == nil
}

// LoadSecret reads a secret saved with StoreSecret.
func LoadSecret(account string) (string, bool) {
	if !keychainOn() {
		return "", false
	}
	var secret string
	err := withTimeout(func() error {
		var err error
		secret, err = keyring.Get(keychainService, account)
		return err
	})
	return secret, err == nil
}

// DeleteSecret removes a secret saved with StoreSecret, if present.
func DeleteSecret(account string) {
	if !keychainOn() {
		return
	}
	_ = withTimeout(func() error { return keyring.Delete(keychainService, account) })
}

// withTimeout keeps a locked or unresponsive keychain (for example a Linux
// secret service waiting for an unlock prompt) from hanging the CLI.
func withTimeout(fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(keychainTimeout):
		keychainHung.Store(true)
		return errKeychainTimeout
	}
}

// UseMockKeychain swaps in an in-memory keychain until restore is called.
// failWith makes every keychain call fail with that error.
func UseMockKeychain(failWith error) (restore func()) {
	if failWith != nil {
		keyring.MockInitWithError(failWith)
	} else {
		keyring.MockInit()
	}
	prev := keychainEnabled
	keychainEnabled = true
	keychainHung.Store(false)
	return func() { keychainEnabled = prev }
}
