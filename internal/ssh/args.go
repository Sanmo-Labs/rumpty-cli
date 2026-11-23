package ssh

import (
	"strconv"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
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
		args = append(args, remoteCommand(opts.Command))
	}
	return args
}

// remoteCommand builds the command string ssh hands to the VM's shell. A
// single argument is used as written, so `rumpty exec vm -- 'ls | wc -l'`
// keeps working as a shell command line. Several arguments are quoted one by
// one, so each reaches the remote program exactly as typed.
func remoteCommand(command []string) string {
	if len(command) == 1 {
		return command[0]
	}
	quoted := make([]string, len(command))
	for i, arg := range command {
		quoted[i] = posixQuote(arg)
	}
	return strings.Join(quoted, " ")
}
