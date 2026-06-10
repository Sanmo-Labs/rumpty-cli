package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

type copySession struct {
	sshBin       string
	proxyCommand string
	session      *api.CertResponse
	paths        CopyPaths
	opts         *Options
	recursive    bool
	keyDir       string
}

// Copy transfers files between the local machine and a workspace VM using rsync,
// falling back to scp when rsync is unavailable locally or on the VM.
func Copy(ctx context.Context, rt *app.Runtime, src, dest string, opts *Options, recursive bool) error {
	paths, err := ParseCopyPaths(src, dest)
	if err != nil {
		return err
	}

	vmRef, err := resolveVMRef(ctx, rt, paths.VMRef)
	if err != nil {
		return err
	}

	if opts == nil || !opts.Debug {
		term.Statusf(stderr(rt), "Preparing file transfer for %s", vmRef)
	}

	key, session, err := obtainSession(ctx, rt, vmRef, opts)
	if err != nil {
		return err
	}

	sshBin, err := SSHBinPath()
	if err != nil {
		return err
	}

	keyPath, certPath, cleanup, err := writeSessionKeys(key, &session)
	if err != nil {
		return err
	}
	defer cleanup()

	debug := opts != nil && opts.Debug
	proxyCommand := buildProxyCommand(sshBin, &session, keyPath, certPath, debug)
	if !debug {
		term.Statusf(stderr(rt), "Transferring files as %s", session.Username)
	}

	return copyWithFallback(ctx, rt, &copySession{
		sshBin:       sshBin,
		proxyCommand: proxyCommand,
		session:      &session,
		paths:        paths,
		opts:         opts,
		recursive:    copyRecursive(paths, recursive),
		keyDir:       filepath.Dir(keyPath),
	})
}

func copyWithFallback(ctx context.Context, rt *app.Runtime, cs *copySession) error {
	fallback, err := tryRsync(ctx, cs)
	if err != nil {
		return err
	}
	if fallback == "" {
		return nil
	}
	warnCopyFallback(rt, fallback)
	return runSCP(ctx, cs)
}

// tryRsync runs rsync when available. An empty fallback reason with nil error means
// success; a non-empty fallback reason means scp should be used instead.
func tryRsync(ctx context.Context, cs *copySession) (string, error) {
	rsyncBin, err := RsyncBinPath()
	if errors.Is(err, ErrRsyncNotFound) {
		return "rsync not found locally", nil
	}
	if err != nil {
		return "", err
	}

	wrapper, err := writeRsyncSSHWrapper(cs.keyDir, cs.sshBin, cs.proxyCommand, cs.opts)
	if err != nil {
		return "", err
	}

	args := buildRsyncArgs(wrapper, cs.session, cs.paths, cs.opts)
	cmd := exec.CommandContext(ctx, rsyncBin, args...)
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Run(); err != nil {
		if rsyncMissingOnRemote(stderrBuf.String()) {
			return "rsync not found on VM", nil
		}
		return "", wrapRunErr(err)
	}
	return "", nil
}

func warnCopyFallback(rt *app.Runtime, reason string) {
	fmt.Fprintf(stderr(rt), "rumpty: %s, falling back to scp\n", reason)
	rumptylog.Warn("copy falling back to scp", "reason", reason)
}

func runSCP(ctx context.Context, cs *copySession) error {
	scpBin, err := SCPBinPath()
	if err != nil {
		return err
	}

	srcPath, destPath, args := buildSCPArgs(cs.proxyCommand, cs.session, cs.paths, cs.opts, cs.recursive)
	args = append(args, srcPath, destPath)
	cmd := exec.CommandContext(ctx, scpBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return wrapRunErr(err)
	}
	return nil
}
