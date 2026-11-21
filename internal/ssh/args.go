package ssh

import (
	"strconv"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

// buildProxyCommand returns the inner ssh invocation used as ProxyCommand for the edge hop.
// OpenSSH runs it through a shell and expands %-tokens in it, so every argument is
// quoted and literal % signs are doubled.
func buildProxyCommand(sshBin string, session *api.CertResponse, keyPath, certPath string, debug bool) string {
	args := []string{sshBin}
	args = append(args, quietSSHOptions(debug)...)
	args = append(args, hostKeyOptions()...)
	args = append(args,
		"-i", keyPath,
		"-o", "CertificateFile="+certPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "RequestTTY=no",
		"-T",
		"-p", strconv.Itoa(session.EdgePort),
		"--",
		session.RouterUser+"@"+session.EdgeHost,
	)
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.ReplaceAll(strings.Join(quoted, " "), "%", "%%")
}

// buildSSHArgs assembles arguments for the outer ssh process (VM hop).
func buildSSHArgs(proxyCommand string, session *api.CertResponse, opts *Options) []string {
	debug := opts != nil && opts.Debug
	args := []string{"-o", "ProxyCommand=" + proxyCommand, "-o", "CheckHostIP=no"}
	args = append(args, quietSSHOptions(debug)...)
	args = append(args, hostKeyOptions()...)
	args = append(args,
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
		"-o", "HostkeyAlgorithms=+ssh-rsa",
	)
	if debug {
		args = append(args, "-vvv")
	}
	if opts.Interactive() {
		args = append(args, "-o", "RequestTTY=yes")
	} else {
		args = append(args, "-T", "-o", "BatchMode=yes")
		if opts != nil && opts.AllocateTTY {
			args = append(args, "-t")
		}
	}
	if opts != nil && strings.TrimSpace(opts.IdentityFile) != "" {
		args = append(args,
			"-i", opts.IdentityFile,
			"-o", "IdentitiesOnly=yes",
		)
	}
	remote := session.Username + "@" + session.VMSlug
	args = append(args, "--", remote)
	if opts != nil && len(opts.Command) > 0 {
		args = append(args, opts.Command...)
	}
	return args
}
