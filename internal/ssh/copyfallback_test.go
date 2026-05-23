package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestRsyncMissingOnRemote(t *testing.T) {
	t.Parallel()

	assert.True(t, rumptyssh.RsyncMissingOnRemoteForTest("bash: line 1: rsync: command not found"))
	assert.True(t, rumptyssh.RsyncMissingOnRemoteForTest("sh: 1: rsync: not found"))
	assert.False(t, rumptyssh.RsyncMissingOnRemoteForTest("Permission denied (13)"))
}

func TestCopyRecursive_uploadDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	paths := rumptyssh.CopyPaths{
		Local:  dir,
		Upload: true,
	}
	assert.True(t, rumptyssh.CopyRecursiveForTest(paths, false))
}

func TestCopyRecursive_downloadNeedsFlag(t *testing.T) {
	t.Parallel()

	paths := rumptyssh.CopyPaths{
		Local:  "./out",
		Upload: false,
	}
	assert.False(t, rumptyssh.CopyRecursiveForTest(paths, false))
	assert.True(t, rumptyssh.CopyRecursiveForTest(paths, true))
}
