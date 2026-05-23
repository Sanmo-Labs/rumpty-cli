package ssh_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestBuildSSHArgs_exec(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{
		Username: "ubuntu",
		VMSlug:   "my-vm",
	}
	opts := &rumptyssh.Options{Command: []string{"echo", "hello"}}
	args := rumptyssh.BuildSSHArgsForTest("ssh -W %h:%p gateway", session, opts)

	assert.Contains(t, args, "-T")
	assert.Contains(t, args, "BatchMode=yes")
	assert.Equal(t, "ubuntu@my-vm", args[len(args)-3])
	assert.Equal(t, "echo", args[len(args)-2])
	assert.Equal(t, "hello", args[len(args)-1])
}

func TestBuildSSHArgs_interactive(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	args := rumptyssh.BuildSSHArgsForTest("proxy", session, &rumptyssh.Options{})

	joined := strings.Join(args, " ")
	assert.Contains(t, args, "RequestTTY=yes")
	assert.Contains(t, joined, "StrictHostKeyChecking=accept-new")
	assert.Contains(t, joined, "UserKnownHostsFile=")
	assert.Contains(t, joined, "LogLevel=ERROR")
	assert.NotContains(t, joined, "BatchMode=yes")
	assert.Equal(t, "ubuntu@my-vm", args[len(args)-1])
}

func TestBuildProxyCommand_quotesPaths(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{
		RouterUser: "ubuntu+my-vm",
		EdgeHost:   "ssh.example.com",
		EdgePort:   22,
	}
	proxy := rumptyssh.BuildProxyCommandForTest("/usr/bin/ssh", session, "/tmp/rumpty ssh/id", "/tmp/rumpty ssh/id-cert.pub")
	assert.Contains(t, proxy, `"/tmp/rumpty ssh/id"`)
	assert.Contains(t, proxy, `CertificateFile=`)
	assert.Contains(t, proxy, "LogLevel=ERROR")
	assert.NotContains(t, proxy, "-vvv")
}

func TestSSHBinPath_orError(t *testing.T) {
	t.Parallel()

	_, err := rumptyssh.SSHBinPath()
	if err != nil {
		require.ErrorIs(t, err, rumptyssh.ErrSSHNotFound)
	}
}
