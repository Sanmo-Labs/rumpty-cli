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

type Options struct {
	GuestUser    string
	IdentityFile string
	Debug        bool
}

func Open(ctx context.Context, rt *app.Runtime, vm string, opts Options) error {
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
		Username:  strings.TrimSpace(opts.GuestUser),
		PublicKey: pubLine,
	})
	if err != nil {
		return err
	}
	rumptylog.Debug("connecting with ssh", "host", session.EdgeHost, "port", session.EdgePort, "router_user", session.RouterUser)
	return Dial(ctx, &session, key, opts)
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

func Dial(ctx context.Context, session *api.CertResponse, key KeyPair, opts Options) error {
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

	proxyCommand := strings.Join([]string{
		"ssh",
		"-i", keyPath,
		"-o", "CertificateFile=" + certPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "RequestTTY=no",
		"-T",
		"-p", strconv.Itoa(session.EdgePort),
		session.RouterUser + "@" + session.EdgeHost,
	}, " ")

	args := []string{
		"-o", "ProxyCommand=" + proxyCommand,
		"-o", "CheckHostIP=no",
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
		"-o", "HostkeyAlgorithms=+ssh-rsa",
	}
	if opts.Debug {
		args = append(args, "-vvv")
	}
	if strings.TrimSpace(opts.IdentityFile) != "" {
		args = append(args,
			"-i", opts.IdentityFile,
			"-o", "IdentitiesOnly=yes",
		)
	}
	args = append(args, session.Username+"@"+session.VMSlug)
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
