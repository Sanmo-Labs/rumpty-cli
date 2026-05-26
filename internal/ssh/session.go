package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
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
	Command      []string
	AllocateTTY  bool
	Stdin        io.Reader
}

func (o *Options) Interactive() bool {
	return o == nil || len(o.Command) == 0
}

func Open(ctx context.Context, rt *app.Runtime, ref string, opts *Options) error {
	vmRef, err := resolveVMRef(ctx, rt, ref)
	if err != nil {
		return err
	}
	return connect(ctx, rt, vmRef, opts)
}

func Exec(ctx context.Context, rt *app.Runtime, ref string, command []string, opts *Options) error {
	vmRef, err := resolveVMRef(ctx, rt, ref)
	if err != nil {
		return err
	}
	if opts == nil {
		opts = &Options{}
	}
	opts.Command = command
	return connect(ctx, rt, vmRef, opts)
}

func connect(ctx context.Context, rt *app.Runtime, vmRef string, opts *Options) error {
	if opts == nil || !opts.Debug {
		term.Statusf(stderr(rt), "Preparing SSH access for %s", vmRef)
	}

	key, session, err := obtainSession(ctx, rt, vmRef, opts)
	if err != nil {
		return err
	}
	rumptylog.Debug("connecting with ssh", "host", session.EdgeHost, "port", session.EdgePort, "router_user", session.RouterUser)
	if opts == nil || !opts.Debug {
		term.Statusf(stderr(rt), "Opening SSH session as %s", session.Username)
	}
	return Dial(ctx, &session, key, opts, nil)
}

func obtainSession(ctx context.Context, rt *app.Runtime, vmRef string, opts *Options) (KeyPair, api.CertResponse, error) {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	apiURL := strings.TrimSpace(rt.Config.APIURL)
	user := guestUser(opts)

	if key, session, ok := sessionCache.get(apiURL, workspace, vmRef, user); ok {
		rumptylog.Debug("reusing cached ssh certificate", "vm", vmRef, "workspace", workspace)
		return key, session, nil
	}

	key, err := NewKeyPair()
	if err != nil {
		return KeyPair{}, api.CertResponse{}, err
	}
	pubLine, err := key.AuthorizedKeyLine()
	if err != nil {
		return KeyPair{}, api.CertResponse{}, err
	}

	rumptylog.Debug("requesting SSH certificate", "vm", vmRef, "workspace", workspace)
	session, err := rt.API().IssueSSHCert(ctx, workspace, api.CertRequest{
		VM:        vmRef,
		Username:  user,
		PublicKey: pubLine,
	})
	if err != nil {
		return KeyPair{}, api.CertResponse{}, err
	}
	sessionCache.put(apiURL, workspace, vmRef, user, key, &session)
	return key, session, nil
}

func stderr(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}

func guestUser(opts *Options) string {
	if opts == nil {
		return ""
	}
	return strings.TrimSpace(opts.GuestUser)
}

func stopSpinner(spin *term.Spinner) {
	if spin != nil {
		spin.Stop()
	}
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

func Dial(ctx context.Context, session *api.CertResponse, key KeyPair, opts *Options, spin *term.Spinner) error {
	defer stopSpinner(spin)

	sshBin, err := SSHBinPath()
	if err != nil {
		return err
	}

	keyPath, certPath, cleanup, err := writeSessionKeys(key, session)
	if err != nil {
		return err
	}
	defer cleanup()

	proxyCommand := buildProxyCommand(sshBin, session, keyPath, certPath, opts != nil && opts.Debug)
	args := buildSSHArgs(proxyCommand, session, opts)

	cmd := exec.CommandContext(ctx, sshBin, args...)
	if opts.Interactive() {
		cmd.Stdin = os.Stdin
	} else if opts != nil && opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	stopSpinner(spin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return wrapRunErr(err)
	}
	return nil
}

func wrapRunErr(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	return err
}

func writeSessionKeys(key KeyPair, session *api.CertResponse) (keyPath, certPath string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "rumpty-ssh-*")
	if err != nil {
		return "", "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	keyPath = filepath.Join(dir, "id_ed25519")
	certPath = keyPath + "-cert.pub"

	block, err := ssh.MarshalPrivateKey(key.Private, "rumpty-temporary")
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("marshal temporary private key: %w", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("write temporary private key: %w", err)
	}
	if err := os.WriteFile(certPath, []byte(session.Certificate), 0o600); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("write temporary certificate: %w", err)
	}
	return keyPath, certPath, cleanup, nil
}
