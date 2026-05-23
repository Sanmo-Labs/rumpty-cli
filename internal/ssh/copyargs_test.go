package ssh_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestBuildRsyncArgs_upload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wrapper, err := rumptyssh.WriteRsyncSSHWrapperForTest(dir, "/usr/bin/ssh", "ssh -W proxy", nil)
	require.NoError(t, err)

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	paths := rumptyssh.CopyPaths{
		VMRef:  "my-vm",
		Local:  "./dist",
		Remote: "/srv/app/",
		Upload: true,
	}
	args := rumptyssh.BuildRsyncArgsForTest(wrapper, session, paths, nil)

	assert.Equal(t, "-a", args[0])
	assert.Equal(t, "-e", args[1])
	assert.Equal(t, wrapper, args[2])
	assert.Equal(t, "./dist", args[3])
	assert.Equal(t, "ubuntu@my-vm:/srv/app/", args[4])

	data, err := os.ReadFile(wrapper)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, strings.HasPrefix(content, "#!/bin/sh\nexec "))
	assert.Contains(t, content, "ProxyCommand=")
	assert.Contains(t, content, `"$@"`)
}

func TestBuildSCPArgs_download(t *testing.T) {
	t.Parallel()

	session := &api.CertResponse{Username: "ubuntu", VMSlug: "my-vm"}
	paths := rumptyssh.CopyPaths{
		VMRef:  "my-vm",
		Local:  "./out",
		Remote: "/tmp/x",
		Upload: false,
	}
	src, dest, args := rumptyssh.BuildSCPArgsForTest("proxy-cmd", session, paths, nil, true)

	assert.Equal(t, "ubuntu@my-vm:/tmp/x", src)
	assert.Equal(t, "./out", dest)
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, "ProxyCommand=proxy-cmd")
	assert.Contains(t, args, "-r")
}

func TestRsyncBinPath_orError(t *testing.T) {
	t.Parallel()

	_, err := rumptyssh.RsyncBinPath()
	if err != nil {
		require.ErrorIs(t, err, rumptyssh.ErrRsyncNotFound)
	}
}

func TestSCPBinPath_orError(t *testing.T) {
	t.Parallel()

	_, err := rumptyssh.SCPBinPath()
	if err != nil {
		require.ErrorIs(t, err, rumptyssh.ErrSCPNotFound)
	}
}
