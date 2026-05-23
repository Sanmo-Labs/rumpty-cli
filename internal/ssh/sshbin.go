package ssh

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// ErrSSHNotFound is returned when the OpenSSH client is not on PATH.
var ErrSSHNotFound = errors.New("openssh client not found")

// SSHBinPath returns the path to the OpenSSH ssh executable, or ErrSSHNotFound.
func SSHBinPath() (string, error) {
	path, err := exec.LookPath("ssh")
	if err != nil {
		return "", fmt.Errorf("%w: install the OpenSSH client and ensure %q is on your PATH (%s)",
			ErrSSHNotFound, "ssh", sshInstallHint())
	}
	return path, nil
}

func sshInstallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "Settings → Apps → Optional features → OpenSSH Client"
	case "darwin":
		return "usually preinstalled; otherwise install via Xcode CLT or Homebrew"
	default:
		return "install the OpenSSH client for your OS (package name varies: openssh-client, openssh, etc.)"
	}
}
