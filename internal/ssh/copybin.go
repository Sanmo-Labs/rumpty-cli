package ssh

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// ErrRsyncNotFound is returned when rsync is not on PATH.
var ErrRsyncNotFound = errors.New("rsync not found")

// ErrSCPNotFound is returned when the OpenSSH scp client is not on PATH.
var ErrSCPNotFound = errors.New("openssh scp client not found")

// RsyncBinPath returns the path to rsync, or ErrRsyncNotFound.
func RsyncBinPath() (string, error) {
	path, err := exec.LookPath("rsync")
	if err != nil {
		return "", fmt.Errorf("%w: install rsync and ensure %q is on your PATH (%s)",
			ErrRsyncNotFound, "rsync", rsyncInstallHint())
	}
	return path, nil
}

// SCPBinPath returns the path to the OpenSSH scp executable, or ErrSCPNotFound.
func SCPBinPath() (string, error) {
	path, err := exec.LookPath("scp")
	if err != nil {
		return "", fmt.Errorf("%w: install the OpenSSH client and ensure %q is on your PATH (%s)",
			ErrSCPNotFound, "scp", scpInstallHint())
	}
	return path, nil
}

func rsyncInstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install rsync, or use the system copy if available"
	case "windows":
		return "install via Chocolatey, Scoop, or WSL"
	default:
		return "install the rsync package for your OS (package name varies)"
	}
}

func scpInstallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "Settings → Apps → Optional features → OpenSSH Client"
	case "darwin":
		return "usually preinstalled with OpenSSH; otherwise install via Xcode CLT or Homebrew"
	default:
		return "install the OpenSSH client for your OS (package name varies: openssh-client, openssh, etc.)"
	}
}
