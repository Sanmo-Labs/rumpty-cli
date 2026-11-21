package credentials_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

func TestWriteSecretFile_replacesLooseFileWithPrivateOne(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "rumpty")
	require.NoError(t, os.MkdirAll(dir, 0o755)) //nolint:gosec // starts loose on purpose
	path := filepath.Join(dir, "secret.json")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	require.NoError(t, credentials.WriteSecretFile(path, []byte("new")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	dirInfo, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary files left behind")
}

func TestLoad_tightensLooseCredentialsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{Token: "tok"}))
	path, err := credentials.Path()
	require.NoError(t, err)
	require.NoError(t, os.Chmod(path, 0o644)) //nolint:gosec // starts loose on purpose

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "tok", creds.Token)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
