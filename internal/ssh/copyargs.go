package ssh

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func buildCopySSHOptions(proxyCommand string, opts *Options) []string {
	debug := opts != nil && opts.Debug
	args := []string{"-o", "ProxyCommand=" + proxyCommand, "-o", "CheckHostIP=no"}
	args = append(args, quietSSHOptions(debug)...)
	args = append(args, hostKeyOptions()...)
	args = append(args,
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
		"-o", "HostkeyAlgorithms=+ssh-rsa",
		"-T",
		"-o", "BatchMode=yes",
		"-o", "RequestTTY=no",
	)
	if debug {
		args = append(args, "-vvv")
	}
	if opts != nil && strings.TrimSpace(opts.IdentityFile) != "" {
		args = append(args,
			"-i", opts.IdentityFile,
			"-o", "IdentitiesOnly=yes",
		)
	}
	return args
}

// writeRsyncSSHWrapper writes a shell script rsync can pass to -e. Rsync runs -e via
// /bin/sh, so ProxyCommand values with spaces must not be assembled into one string.
func writeRsyncSSHWrapper(dir, sshBin, proxyCommand string, opts *Options) (string, error) {
	var b strings.Builder
	b.WriteString("#!/bin/sh\nexec ")
	b.WriteString(shellQuote(sshBin))
	for _, arg := range buildCopySSHOptions(proxyCommand, opts) {
		b.WriteString(" ")
		b.WriteString(shellQuote(arg))
	}
	b.WriteString(` "$@"`)
	b.WriteByte('\n')

	path := filepath.Join(dir, "rumpty-rsync-ssh")
	if err := os.WriteFile(path, []byte(b.String()), 0o700); err != nil {
		return "", err
	}
	return path, nil
}

func buildRsyncArgs(rsyncSSHWrapper string, session *api.CertResponse, paths CopyPaths, opts *Options) []string {
	remote := remoteCopyTarget(session, paths.Remote)
	args := []string{"-a", "-e", rsyncSSHWrapper}
	if paths.Upload {
		args = append(args, paths.Local, remote)
	} else {
		args = append(args, remote, paths.Local)
	}
	return args
}

func buildSCPArgs(proxyCommand string, session *api.CertResponse, paths CopyPaths, opts *Options, recursive bool) (src, dest string, args []string) {
	remote := remoteCopyTarget(session, paths.Remote)
	args = buildCopySSHOptions(proxyCommand, opts)
	if recursive {
		args = append(args, "-r")
	}
	if paths.Upload {
		return paths.Local, remote, args
	}
	return remote, paths.Local, args
}
