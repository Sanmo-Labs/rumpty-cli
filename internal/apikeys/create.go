package apikeys

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// CreateOpts carries the flags for creating an API key.
type CreateOpts struct {
	ExpiresDays int
	NeverExpiry bool
	Scope       string
	Permissions string
}

// Create makes a new personal API key and prints its one-time token.
func Create(ctx context.Context, rt *app.Runtime, name string, opts CreateOpts) error {
	if name == "" {
		return config.NewUsageError("a key name is required")
	}
	if opts.NeverExpiry && opts.ExpiresDays > 0 {
		return config.NewUsageError("--never-expires and --expires-days cannot be combined")
	}
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != "" && scope != "cli" && scope != "mcp" {
		return config.NewUsageError("--scope must be cli or mcp")
	}
	var permissions []string
	if strings.TrimSpace(opts.Permissions) != "" {
		for _, p := range strings.Split(opts.Permissions, ",") {
			if p = strings.TrimSpace(p); p != "" {
				permissions = append(permissions, p)
			}
		}
	}

	spin := term.StartLoading(errOut(rt), "Creating API key "+name)
	result, err := rt.API().CreateAPIKey(ctx, api.CreateAPIKeyRequest{
		Name:          name,
		ExpiresInDays: opts.ExpiresDays,
		NeverExpires:  opts.NeverExpiry,
		Scope:         scope,
		Permissions:   permissions,
	})
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Created API key %s (prefix %s, scope %s)", result.Name, result.Prefix, scopeOf(result.Scopes))))
	fmt.Fprintln(rt.Streams.Out, term.Muted(errOut(rt), "The token is shown only once; store it securely, it cannot be retrieved again"))
	fmt.Fprintln(rt.Streams.Out, result.Token)
	return nil
}

func scopeOf(scopes []string) string {
	if len(scopes) == 0 {
		return "cli"
	}
	return strings.Join(scopes, ", ")
}
