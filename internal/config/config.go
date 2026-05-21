package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

const (
	EnvAPIURL    = "RUMPTY_API_URL"
	EnvToken     = "RUMPTY_API_KEY"
	EnvWorkspace = "RUMPTY_WORKSPACE"
	EnvLogLevel  = "RUMPTY_LOG_LEVEL"
)

var (
	DefaultAPIURL = "http://localhost:8889"
)

type Config struct {
	APIURL    string
	Token     string
	Workspace string
	LogLevel  string
	Verbose   bool
}

// LogLevelValue returns the effective log level from --verbose, --log-level, or $RUMPTY_LOG_LEVEL.
func (c *Config) LogLevelValue() string {
	if c.Verbose {
		return "debug"
	}
	if v := strings.TrimSpace(c.LogLevel); v != "" {
		return v
	}
	return envOr(EnvLogLevel, "warn")
}

func (c *Config) Resolve() {
	creds, _ := credentials.Load()

	if strings.TrimSpace(c.APIURL) == "" {
		c.APIURL = envOr(EnvAPIURL, creds.APIURL)
	}
	if strings.TrimSpace(c.APIURL) == "" {
		c.APIURL = DefaultAPIURL
	}
	if strings.TrimSpace(c.Token) == "" {
		c.Token = envOr(EnvToken, creds.Token)
	}
	if strings.TrimSpace(c.Workspace) == "" {
		c.Workspace = strings.TrimSpace(os.Getenv(EnvWorkspace))
	}
}

func (c *Config) ValidateForAuth() error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("missing rumpty login or $%s", EnvToken)
	}
	return nil
}

func (c *Config) ValidateForSSH() error {
	var missing []string
	if strings.TrimSpace(c.Token) == "" {
		missing = append(missing, "rumpty login or $"+EnvToken)
	}
	if strings.TrimSpace(c.Workspace) == "" {
		missing = append(missing, "$"+EnvWorkspace+", --ws, or --workspace")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing %s", strings.Join(missing, ", "))
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type UsageError struct {
	Message string
}

func (e *UsageError) Error() string { return e.Message }

func NewUsageError(format string, args ...any) error {
	return &UsageError{Message: fmt.Sprintf(format, args...)}
}

func IsUsageError(err error) bool {
	var u *UsageError
	return errors.As(err, &u)
}
