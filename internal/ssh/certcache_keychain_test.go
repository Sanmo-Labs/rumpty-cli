package ssh_test

import (
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
	rumptyssh "github.com/cloverworks-hq/rumpty-cli/internal/ssh"
)

func cachedSessionExpiringIn(d time.Duration) api.CertResponse {
	return api.CertResponse{
		EdgeHost:    "ssh.example.com",
		RouterUser:  "ubuntu+vm",
		Certificate: "cert",
		ExpiresAt:   time.Now().Add(d).UTC().Format(time.RFC3339),
	}
}

func certStoreFile(t *testing.T) string {
	t.Helper()
	path, err := rumptyssh.CertStorePathForTest()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestCertCache_privateKeyLivesInKeychain(t *testing.T) {
	resetCertCaches(t)
	defer credentials.UseMockKeychain(nil)()

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := cachedSessionExpiringIn(2 * time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)

	stored := certStoreFile(t)
	assert.NotContains(t, stored, `"private_key":`)
	assert.Contains(t, stored, `"private_key_in_keychain": true`)

	rumptyssh.ResetCertCacheForTest()
	gotKey, _, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	require.True(t, ok)
	assert.Equal(t, key.Private, gotKey.Private)
}

func TestCertCache_missingKeychainKeyIsACacheMiss(t *testing.T) {
	resetCertCaches(t)
	defer credentials.UseMockKeychain(nil)()

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := cachedSessionExpiringIn(2 * time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)
	credentials.DeleteSecret(rumptyssh.CertKeyAccountForTest(testAPIURL, "acme", "my-vm", ""))

	rumptyssh.ResetCertCacheForTest()
	_, _, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	assert.False(t, ok)
}

func TestCertCache_expiredAndClearedKeysLeaveKeychain(t *testing.T) {
	resetCertCaches(t)
	defer credentials.UseMockKeychain(nil)()

	oldKey, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	expired := cachedSessionExpiringIn(-time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "old-vm", "", oldKey, &expired)
	oldAccount := rumptyssh.CertKeyAccountForTest(testAPIURL, "acme", "old-vm", "")
	_, ok := credentials.LoadSecret(oldAccount)
	require.True(t, ok)

	newKey, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	fresh := cachedSessionExpiringIn(2 * time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "new-vm", "", newKey, &fresh)
	_, ok = credentials.LoadSecret(oldAccount)
	assert.False(t, ok, "an expired entry's key is removed from the keychain when the cache is pruned")

	newAccount := rumptyssh.CertKeyAccountForTest(testAPIURL, "acme", "new-vm", "")
	require.NoError(t, rumptyssh.ClearCertCache())
	_, ok = credentials.LoadSecret(newAccount)
	assert.False(t, ok, "logout removes cached keys from the keychain")
}

func TestCertCache_keyFromAnotherProcessIsACacheMiss(t *testing.T) {
	resetCertCaches(t)
	defer credentials.UseMockKeychain(nil)()

	mine, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := cachedSessionExpiringIn(2 * time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", mine, &session)

	theirs, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	require.True(t, credentials.StoreSecret(rumptyssh.CertKeyAccountForTest(testAPIURL, "acme", "my-vm", ""), base64.StdEncoding.EncodeToString(theirs.Private)))

	rumptyssh.ResetCertCacheForTest()
	_, _, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	assert.False(t, ok, "a key that does not match the cached certificate must not be reused")
}

func TestCertCache_withoutKeychainKeyStaysInPrivateFile(t *testing.T) {
	resetCertCaches(t)
	t.Setenv(credentials.EnvNoKeychain, "1")

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := cachedSessionExpiringIn(2 * time.Minute)
	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)

	assert.Contains(t, certStoreFile(t), `"private_key":`)
	path, err := rumptyssh.CertStorePathForTest()
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	rumptyssh.ResetCertCacheForTest()
	gotKey, _, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	require.True(t, ok)
	assert.Equal(t, key.Private, gotKey.Private)
}
