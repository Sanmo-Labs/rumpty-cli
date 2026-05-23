package ssh_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestKnownHostsFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	path, err := rumptyssh.KnownHostsFileForTest()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "rumpty", "known_hosts"), path)
}
