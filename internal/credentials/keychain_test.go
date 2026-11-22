package credentials_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
)

func readCredentialsFile(t *testing.T) string {
	t.Helper()
	path, err := credentials.Path()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestSave_keepsTokenInKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer credentials.UseMockKeychain(nil)()

	require.NoError(t, credentials.Save(credentials.File{APIURL: "https://api.example", Token: "secret-token", Username: "alice"}))
	assert.NotContains(t, readCredentialsFile(t), "secret-token")

	stored, ok := credentials.LoadSecret("api-key")
	require.True(t, ok)
	assert.Equal(t, "secret-token", stored)

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "secret-token", creds.Token)
	assert.Equal(t, "alice", creds.Username)

	require.NoError(t, credentials.Clear())
	_, ok = credentials.LoadSecret("api-key")
	assert.False(t, ok)
}

func TestLoad_movesFileTokenIntoKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(credentials.EnvNoKeychain, "1")
	require.NoError(t, credentials.Save(credentials.File{Token: "old-token"}))
	require.Contains(t, readCredentialsFile(t), "old-token")

	t.Setenv(credentials.EnvNoKeychain, "")
	defer credentials.UseMockKeychain(nil)()

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "old-token", creds.Token)
	assert.NotContains(t, readCredentialsFile(t), "old-token")

	stored, ok := credentials.LoadSecret("api-key")
	require.True(t, ok)
	assert.Equal(t, "old-token", stored)
}

func TestSave_fallsBackToFileWithoutKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer credentials.UseMockKeychain(errors.New("no secret service"))()

	require.NoError(t, credentials.Save(credentials.File{Token: "file-token"}))
	assert.Contains(t, readCredentialsFile(t), "file-token")

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "file-token", creds.Token)
}

func TestSave_optOutKeepsTokenInFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(credentials.EnvNoKeychain, "1")
	defer credentials.UseMockKeychain(nil)()

	require.NoError(t, credentials.Save(credentials.File{Token: "file-token"}))
	assert.Contains(t, readCredentialsFile(t), "file-token")
}
