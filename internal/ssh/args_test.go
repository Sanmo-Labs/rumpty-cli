package ssh_test

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	rumptyssh "github.com/cloverworks-hq/rumpty-cli/internal/ssh"
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
	assert.Equal(t, "ubuntu@my-vm", args[len(args)-2])
	assert.Equal(t, "'echo' 'hello'", args[len(args)-1])
}

func TestBuildSSHArgs_execSingleArgumentIsAShellCommand(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	opts := &rumptyssh.Options{Command: []string{"ls -la | wc -l"}}
	args := rumptyssh.BuildSSHArgsForTest("proxy", session, opts)
	assert.Equal(t, "ls -la | wc -l", args[len(args)-1])
}

func TestBuildSSHArgs_execKeepsArgumentsAsTyped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting")
	}
	t.Parallel()

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	opts := &rumptyssh.Options{Command: []string{"printf", `[%s]\n`, "a b", "$HOME", "it's"}}
	args := rumptyssh.BuildSSHArgsForTest("proxy", session, opts)

	// The VM's shell sees the same string a remote sh -c would.
	out, err := exec.CommandContext(t.Context(), "/bin/sh", "-c", args[len(args)-1]).Output() //nolint:gosec // runs the generated command on purpose
	require.NoError(t, err)
	assert.Equal(t, "[a b]\n[$HOME]\n[it's]\n", string(out))
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
	assert.NotContains(t, args, "-q")
	assert.NotContains(t, joined, "BatchMode=yes")
	assert.Equal(t, "ubuntu@my-vm", args[len(args)-1])
	assert.Equal(t, "--", args[len(args)-2])
}

func TestBuildProxyCommand_quotesPaths(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{
		RouterUser: "ubuntu+my-vm",
		EdgeHost:   "ssh.example.com",
		EdgePort:   22,
	}
	proxy := rumptyssh.BuildProxyCommandForTest("/usr/bin/ssh", session, "/tmp/rumpty ssh/id", "/tmp/rumpty ssh/id-cert.pub")
	assert.Contains(t, proxy, `'/tmp/rumpty ssh/id'`)
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

func TestSSHArgs_doNotEnableSHA1RSA(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	_, _, scpArgs := rumptyssh.BuildSCPArgsForTest("proxy", session, rumptyssh.CopyPaths{VMRef: "my-vm", Local: "a", Remote: "/b"}, nil, false)
	wrapper, err := rumptyssh.WriteRsyncSSHWrapperForTest(t.TempDir(), "ssh", "proxy", nil)
	require.NoError(t, err)
	script, err := os.ReadFile(wrapper)
	require.NoError(t, err)
	assert.NotContains(t, string(script), "ssh-rsa")
	for name, args := range map[string][]string{
		"exec":        rumptyssh.BuildSSHArgsForTest("proxy", session, &rumptyssh.Options{Command: []string{"true"}}),
		"interactive": rumptyssh.BuildSSHArgsForTest("proxy", session, &rumptyssh.Options{}),
		"scp":         scpArgs,
	} {
		assert.NotContains(t, strings.Join(args, " "), "ssh-rsa", name)
	}
}
