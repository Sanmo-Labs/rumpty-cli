package ssh_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

const testAPIURL = "https://api.example"

func TestCertCache_reuseUntilExpiry(t *testing.T) {
	resetCertCaches(t)
	t.Cleanup(func() { resetCertCaches(t) })

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := api.CertResponse{
		EdgeHost:    "ssh.example.com",
		RouterUser:  "ubuntu+vm",
		Certificate: "cert",
		ExpiresAt:   time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
	}

	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)

	gotKey, gotSession, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	require.True(t, ok)
	assert.Equal(t, session.EdgeHost, gotSession.EdgeHost)
	assert.Equal(t, key.Public, gotKey.Public)
}

func TestCertCache_missAfterExpiry(t *testing.T) {
	resetCertCaches(t)
	t.Cleanup(func() { resetCertCaches(t) })

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := api.CertResponse{
		EdgeHost:    "ssh.example.com",
		RouterUser:  "ubuntu+vm",
		Certificate: "cert",
		ExpiresAt:   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	}

	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)

	_, _, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	assert.False(t, ok)
}

func TestCertCache_persistsAcrossProcessMemoryReset(t *testing.T) {
	resetCertCaches(t)
	t.Cleanup(func() { resetCertCaches(t) })

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)
	session := api.CertResponse{
		EdgeHost:    "ssh.example.com",
		RouterUser:  "ubuntu+vm",
		Certificate: "cert",
		ExpiresAt:   time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
	}

	rumptyssh.PutCertCacheForTest(testAPIURL, "acme", "my-vm", "", key, &session)
	rumptyssh.ResetCertCacheForTest()

	_, gotSession, ok := rumptyssh.GetCertCacheForTest(testAPIURL, "acme", "my-vm", "")
	require.True(t, ok)
	assert.Equal(t, "ssh.example.com", gotSession.EdgeHost)
}

func resetCertCaches(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rumptyssh.ResetCertCacheForTest()
	require.NoError(t, rumptyssh.ResetCertStoreForTest())
}
