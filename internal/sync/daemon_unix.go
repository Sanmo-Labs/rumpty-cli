//go:build !windows

package sync

import (
	"errors"
	"os/exec"
	"syscall"
)

func applyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func killProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
