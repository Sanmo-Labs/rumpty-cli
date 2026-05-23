package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

func knownHostsFile() (string, error) {
	dir, err := credentials.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create rumpty config dir: %w", err)
	}
	return filepath.Join(dir, "known_hosts"), nil
}

func hostKeyOptions() []string {
	path, err := knownHostsFile()
	if err != nil {
		return nil
	}
	return []string{
		"-o", "UserKnownHostsFile=" + path,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UpdateHostKeys=yes",
	}
}

func quietSSHOptions(debug bool) []string {
	if debug {
		return nil
	}
	return []string{
		"-q",
		"-o", "LogLevel=ERROR",
	}
}
