package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
)

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("ssh exited with status %d", e.Code)
}

type KeyPair struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

func Open(ctx context.Context, rt *app.Runtime, vm, guestUser string) error {
	key, err := NewKeyPair()
	if err != nil {
		return err
	}
	pubLine, err := key.AuthorizedKeyLine()
	if err != nil {
		return err
	}

	rumptylog.Debug("requesting SSH certificate", "vm", vm, "workspace", rt.Config.Workspace)
	session, err := rt.API().IssueSSHCert(ctx, rt.Config.Workspace, api.CertRequest{
		VM:        vm,
		Username:  strings.TrimSpace(guestUser),
		PublicKey: pubLine,
	})
	if err != nil {
		return err
	}
	rumptylog.Debug("connecting with ssh", "host", session.EdgeHost, "port", session.EdgePort)
	return Dial(ctx, &session, key)
}

func NewKeyPair() (KeyPair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generate temporary ssh key: %w", err)
	}
	return KeyPair{Public: public, Private: private}, nil
}

func (k KeyPair) AuthorizedKeyLine() (string, error) {
	sshPublicKey, err := ssh.NewPublicKey(k.Public)
	if err != nil {
		return "", fmt.Errorf("encode temporary ssh public key: %w", err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))), nil
}

func Dial(ctx context.Context, session *api.CertResponse, key KeyPair) error {
	dir, err := os.MkdirTemp("", "rumpty-ssh-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	keyPath := filepath.Join(dir, "id_ed25519")
	certPath := keyPath + "-cert.pub"

	block, err := ssh.MarshalPrivateKey(key.Private, "rumpty-temporary")
	if err != nil {
		return fmt.Errorf("marshal temporary private key: %w", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("write temporary private key: %w", err)
	}
	if err := os.WriteFile(certPath, []byte(session.Certificate), 0o600); err != nil {
		return fmt.Errorf("write temporary certificate: %w", err)
	}

	args := []string{
		"-i", keyPath,
		"-o", "CertificateFile=" + certPath,
		"-o", "IdentitiesOnly=yes",
		"-p", strconv.Itoa(session.EdgePort),
		session.RouterUser + "@" + session.EdgeHost,
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return &ExitError{Code: status.ExitStatus()}
			}
		}
		return err
	}
	return nil
}
