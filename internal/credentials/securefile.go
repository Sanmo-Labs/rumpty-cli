package credentials

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteSecretFile writes data to path readable only by the current user.
// It writes a temporary file in the same directory and renames it into
// place, so readers never see a partial file and an existing file with
// looser permissions is replaced rather than reused.
func WriteSecretFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := EnsurePrivateDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path) //nolint:gosec // both paths are in the rumpty config directory
}

// EnsurePrivateDir creates dir if needed and makes it accessible only by the
// current user.
func EnsurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(dir, 0o700) //nolint:gosec // directories need the execute bit
}

// RestrictPermissions removes group and other access from an existing file.
func RestrictPermissions(path string) {
	if runtime.GOOS == "windows" {
		return
	}
	if info, err := os.Stat(path); err == nil && info.Mode().Perm()&0o077 != 0 {
		_ = os.Chmod(path, info.Mode().Perm()&0o700)
	}
}
