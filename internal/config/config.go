package config

import (
	"errors"
	"fmt"
	"net/url"
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
	DefaultAPIURL = "https://api.rumptycloud.com"
)

type Config struct {
	APIURL    string
	Token     string
	Workspace string
	LogLevel  string
	Verbose   bool

	savedLoginURL string
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
		c.Token = strings.TrimSpace(os.Getenv(EnvToken))
	}
	if strings.TrimSpace(c.Token) == "" && strings.TrimSpace(creds.Token) != "" {
		// A saved login only authenticates against the API it was created for.
		if saved := SavedAPIURL(creds); SameAPIURL(c.APIURL, saved) {
			c.Token = creds.Token
		} else {
			c.savedLoginURL = saved
		}
	}
	if strings.TrimSpace(c.Workspace) == "" {
		c.Workspace = strings.TrimSpace(os.Getenv(EnvWorkspace))
	}
}

// SavedAPIURL returns the API URL a saved login belongs to.
func SavedAPIURL(creds credentials.File) string {
	if u := strings.TrimSpace(creds.APIURL); u != "" {
		return u
	}
	return DefaultAPIURL
}

// SameAPIURL reports whether two API base URLs point at the same API.
func SameAPIURL(a, b string) bool {
	return normalizeAPIURL(a) == normalizeAPIURL(b)
}

func normalizeAPIURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

func (c *Config) missingToken() string {
	if c.savedLoginURL != "" {
		return fmt.Sprintf("$%s (your saved login is for %s, not %s)", EnvToken, c.savedLoginURL, c.APIURL)
	}
	return "rumpty login or $" + EnvToken
}

func (c *Config) ValidateForAuth() error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("missing %s", c.missingToken())
	}
	return nil
}

func (c *Config) ValidateForSSH() error {
	var missing []string
	if strings.TrimSpace(c.Token) == "" {
		missing = append(missing, c.missingToken())
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
