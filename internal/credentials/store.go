package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileName = "credentials.json"

type File struct {
	APIURL          string `json:"api_url,omitempty"`
	Token           string `json:"token,omitempty"`
	TokenInKeychain bool   `json:"token_in_keychain,omitempty"`
	Username        string `json:"username,omitempty"`
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Dir() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "rumpty"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "rumpty"), nil
}

func Load() (File, error) {
	path, err := Path()
	if err != nil {
		return File{}, err
	}

	RestrictPermissions(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("read credentials: %w", err)
	}

	var creds File
	if err := json.Unmarshal(data, &creds); err != nil {
		return File{}, fmt.Errorf("parse credentials: %w", err)
	}
	switch {
	case creds.TokenInKeychain:
		creds.Token, _ = LoadSecret(apiKeyAccount)
	case creds.Token != "" && KeychainAvailable():
		// Move a token saved by an older version into the keychain.
		_ = Save(creds)
	}
	creds.TokenInKeychain = false
	return creds, nil
}

// Save writes creds, keeping the token in the OS keychain when one is
// available and in the file otherwise.
func Save(creds File) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	creds.TokenInKeychain = false
	if creds.Token != "" && StoreSecret(apiKeyAccount, creds.Token) {
		creds.Token = ""
		creds.TokenInKeychain = true
	} else {
		DeleteSecret(apiKeyAccount)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := WriteSecretFile(filepath.Join(dir, fileName), data); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

func Clear() error {
	DeleteSecret(apiKeyAccount)
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
