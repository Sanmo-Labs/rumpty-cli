package credentials_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

func TestDir_usesXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir, err := credentials.Dir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "rumpty"), dir)
}

func TestPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := credentials.Path()
	require.NoError(t, err)
	assert.Equal(t, "credentials.json", filepath.Base(path))
}

func TestLoad_missingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, credentials.File{}, got)
}

func TestLoad_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "rumpty"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rumpty", "credentials.json"), []byte("{"), 0o600))

	_, err := credentials.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse credentials")
}

func TestSaveLoadClear_roundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := credentials.File{
		APIURL:   "https://api.example",
		Token:    "jwt",
		Username: "alice",
	}
	require.NoError(t, credentials.Save(want))

	got, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, want, got)

	require.NoError(t, credentials.Clear())
	got, err = credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, credentials.File{}, got)
}

func TestClear_idempotent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Clear())
	require.NoError(t, credentials.Clear())
}
