package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestParseCopyPaths_upload(t *testing.T) {
	t.Parallel()

	paths, err := rumptyssh.ParseCopyPaths("./local.tar.gz", "my-vm:/tmp/")
	require.NoError(t, err)
	assert.True(t, paths.Upload)
	assert.Equal(t, "my-vm", paths.VMRef)
	assert.Equal(t, "./local.tar.gz", paths.Local)
	assert.Equal(t, "/tmp/", paths.Remote)
}

func TestParseCopyPaths_download(t *testing.T) {
	t.Parallel()

	paths, err := rumptyssh.ParseCopyPaths("my-vm:/var/log/app.log", "./logs/")
	require.NoError(t, err)
	assert.False(t, paths.Upload)
	assert.Equal(t, "my-vm", paths.VMRef)
	assert.Equal(t, "./logs/", paths.Local)
	assert.Equal(t, "/var/log/app.log", paths.Remote)
}

func TestParseCopyPaths_windowsDriveIsLocal(t *testing.T) {
	t.Parallel()

	_, err := rumptyssh.ParseCopyPaths(`C:\data`, `D:\backup`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote path")
}

func TestParseCopyPaths_bothRemote(t *testing.T) {
	t.Parallel()

	_, err := rumptyssh.ParseCopyPaths("vm-a:/a", "vm-b:/b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one")
}
