package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

func runMCP(t *testing.T, args ...string) (string, error) {
	t.Helper()
	rt, _ := newRuntime(t, &config.Config{})
	var out bytes.Buffer
	rt.Streams.Out = &out
	cmd := newMCPCmd(rt)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

func TestMCPClientSnippetsAreValid(t *testing.T) {
	for _, c := range mcpClients {
		t.Run(c.ID, func(t *testing.T) {
			assert.Contains(t, c.Setup, mcpURL)
			assert.NotEmpty(t, c.Check)
			lines := strings.Split(c.Setup, "\n")
			for i, line := range lines {
				if line != "{" {
					continue
				}
				for j := i + 1; j < len(lines); j++ {
					if lines[j] == "}" {
						block := strings.Join(lines[i:j+1], "\n")
						assert.True(t, json.Valid([]byte(block)), "invalid JSON in %s setup:\n%s", c.ID, block)
						break
					}
				}
			}
		})
	}
}

func TestMCPNeverPrintsTheKey(t *testing.T) {
	isolateCreds(t)
	t.Setenv(config.EnvToken, "rumpty_secret_value")
	for _, args := range [][]string{nil, {"cursor"}, {"claude-code"}, {"-o", "json"}, {"zed", "-o", "json"}} {
		out, err := runMCP(t, args...)
		require.NoError(t, err)
		assert.NotContains(t, out, "rumpty_secret_value", "args %v", args)
	}
}

func TestMCPOverviewAndJSON(t *testing.T) {
	isolateCreds(t)
	out, err := runMCP(t)
	require.NoError(t, err)
	assert.Contains(t, out, mcpURL)
	assert.Contains(t, out, mcpKeysURL, "points to where keys are made when none is set")
	for _, c := range mcpClients {
		assert.Contains(t, out, "rumpty mcp "+c.ID)
	}

	out, err = runMCP(t, "Cursor", "-o", "json")
	require.NoError(t, err)
	var info mcpInfo
	require.NoError(t, json.Unmarshal([]byte(out), &info))
	assert.Equal(t, mcpURL, info.URL)
	assert.False(t, info.KeySet)
	require.Len(t, info.Clients, 1)
	assert.Equal(t, "cursor", info.Clients[0].ID)
}

func TestMCPRejectsUnknownClient(t *testing.T) {
	isolateCreds(t)
	_, err := runMCP(t, "notepad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude-code")
}

func TestMCPKeyStepMatchesHowTheClientGetsTheKey(t *testing.T) {
	isolateCreds(t)
	for _, c := range mcpClients {
		out, err := runMCP(t, c.ID)
		require.NoError(t, err)
		assert.Equal(t, c.UsesEnv, strings.Contains(out, "export "+config.EnvToken), "%s", c.ID)
		assert.Equal(t, c.UsesEnv, strings.Contains(c.Setup, config.EnvToken), "%s setup and UsesEnv disagree", c.ID)
	}
}
