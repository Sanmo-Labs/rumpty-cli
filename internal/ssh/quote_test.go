package ssh_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	rumptyssh "github.com/cloverworks-hq/rumpty-cli/internal/ssh"
)

var hostileStrings = []string{
	"simple",
	"with space",
	"$(touch pwned)",
	"`touch pwned`",
	"$HOME",
	"it's",
	`back\slash`,
	"semi;colon",
	"pipe|and&",
	"new\nline",
	"50% off",
	"",
}

func TestPosixQuote_shellReturnsInputUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting")
	}
	t.Parallel()

	for _, in := range hostileStrings {
		out, err := exec.CommandContext(t.Context(), "/bin/sh", "-c", "printf %s "+rumptyssh.PosixQuoteForTest(in)).Output() //nolint:gosec // runs hostile input through sh on purpose
		require.NoError(t, err, in)
		assert.Equal(t, in, string(out))
	}
}

func TestBuildProxyCommand_shellPassesArgumentsUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting")
	}

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "conf $(touch pwned)"))
	printArgs := filepath.Join(dir, "print args")
	require.NoError(t, os.WriteFile(printArgs, []byte("#!/bin/sh\nfor a; do printf '%s\\n' \"$a\"; done\n"), 0o700))

	session := &api.CertResponse{RouterUser: "ubuntu+my-vm", EdgeHost: "ssh.example.com", EdgePort: 2222}
	keyPath := filepath.Join(dir, "key $(touch pwned) 100%")
	certPath := keyPath + "-cert.pub"
	proxy := rumptyssh.BuildProxyCommandForTest(printArgs, session, keyPath, certPath)

	// ssh turns %% back into % before handing the command to the shell.
	cmd := exec.CommandContext(t.Context(), "/bin/sh", "-c", strings.ReplaceAll(proxy, "%%", "%")) //nolint:gosec // runs the generated ProxyCommand through sh on purpose
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)

	args := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	assert.Contains(t, args, keyPath)
	assert.Contains(t, args, "CertificateFile="+certPath)
	assert.Equal(t, []string{"-p", "2222", "--", "ubuntu+my-vm@ssh.example.com"}, args[len(args)-4:])
	assert.NoFileExists(t, filepath.Join(dir, "pwned"))
}

func TestValidateSession(t *testing.T) {
	t.Parallel()

	valid := api.CertResponse{
		Username:   "ubuntu",
		VMSlug:     "my-vm-1",
		RouterUser: "ubuntu+my-vm-1",
		EdgeHost:   "ssh.rumptycloud.com",
		EdgePort:   22,
	}
	require.NoError(t, rumptyssh.ValidateSessionForTest(&valid))

	ipv4 := valid
	ipv4.EdgeHost = "10.0.0.5"
	require.NoError(t, rumptyssh.ValidateSessionForTest(&ipv4))

	tests := []struct {
		name   string
		mutate func(*api.CertResponse)
	}{
		{"host with command", func(s *api.CertResponse) { s.EdgeHost = "x;touch pwned" }},
		{"host with substitution", func(s *api.CertResponse) { s.EdgeHost = "$(id)" }},
		{"host as option", func(s *api.CertResponse) { s.EdgeHost = "-oProxyCommand=id" }},
		{"empty host", func(s *api.CertResponse) { s.EdgeHost = "" }},
		{"router user with space", func(s *api.CertResponse) { s.RouterUser = "a b" }},
		{"router user with backtick", func(s *api.CertResponse) { s.RouterUser = "`id`" }},
		{"username as option", func(s *api.CertResponse) { s.Username = "-oProxyCommand=id" }},
		{"username with at", func(s *api.CertResponse) { s.Username = "a@b" }},
		{"slug as option", func(s *api.CertResponse) { s.VMSlug = "-F/tmp/x" }},
		{"slug with path", func(s *api.CertResponse) { s.VMSlug = "vm/../x" }},
		{"port zero", func(s *api.CertResponse) { s.EdgePort = 0 }},
		{"port too large", func(s *api.CertResponse) { s.EdgePort = 70000 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := valid
			tt.mutate(&s)
			assert.Error(t, rumptyssh.ValidateSessionForTest(&s))
		})
	}
}
