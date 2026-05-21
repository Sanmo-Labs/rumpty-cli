package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

func TestConfig_LogLevelValue(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		env  map[string]string
		want string
	}{
		{name: "verbose forces debug", cfg: config.Config{Verbose: true}, want: "debug"},
		{name: "flag wins over env", cfg: config.Config{LogLevel: "info"}, env: map[string]string{config.EnvLogLevel: "error"}, want: "info"},
		{name: "env when flag empty", env: map[string]string{config.EnvLogLevel: "debug"}, want: "debug"},
		{name: "default warn", want: "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tt.want, tt.cfg.LogLevelValue())
		})
	}
}

func TestConfig_Resolve_preservesExplicitFields(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(config.EnvAPIURL, "https://env.example")
	t.Setenv(config.EnvToken, "env-token")

	cfg := config.Config{
		APIURL:    "https://flag.example",
		Token:     "flag-token",
		Workspace: "ws-flag",
	}
	cfg.Resolve()
	assert.Equal(t, "https://flag.example", cfg.APIURL)
	assert.Equal(t, "flag-token", cfg.Token)
	assert.Equal(t, "ws-flag", cfg.Workspace)
}

func TestConfig_Resolve_fillsFromEnvAndCreds(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{
		APIURL: "https://creds.example",
		Token:  "creds-token",
	}))
	t.Setenv(config.EnvAPIURL, "https://env.example")
	t.Setenv(config.EnvToken, "env-token")
	t.Setenv(config.EnvWorkspace, "ws-env")

	cfg := config.Config{}
	cfg.Resolve()
	assert.Equal(t, "https://env.example", cfg.APIURL)
	assert.Equal(t, "env-token", cfg.Token)
	assert.Equal(t, "ws-env", cfg.Workspace)
}

func TestConfig_Resolve_defaultAPIURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(config.EnvAPIURL, "")
	t.Setenv(config.EnvToken, "")

	cfg := config.Config{}
	cfg.Resolve()
	assert.Equal(t, config.DefaultAPIURL, cfg.APIURL)
}

func TestConfig_ValidateForSSH(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name: "ok when token and workspace set",
			cfg:  config.Config{Token: "tok", Workspace: "acme"},
		},
		{
			name:    "missing token",
			cfg:     config.Config{Workspace: "acme"},
			wantErr: "rumpty login",
		},
		{
			name:    "missing workspace",
			cfg:     config.Config{Token: "tok"},
			wantErr: "RUMPTY_WORKSPACE",
		},
		{
			name:    "missing both",
			cfg:     config.Config{},
			wantErr: "rumpty login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateForSSH()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestUsageError(t *testing.T) {
	t.Parallel()

	err := config.NewUsageError("need %s", "workspace")
	assert.Equal(t, "need workspace", err.Error())
	assert.True(t, config.IsUsageError(err))
	assert.False(t, config.IsUsageError(assert.AnError))
}
