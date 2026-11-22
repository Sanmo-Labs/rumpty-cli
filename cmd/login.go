package commands

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/auth"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	rumptyterm "github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newLoginCmd(rt *app.Runtime) *cobra.Command {
	var apiKey string
	var tokenStdin bool
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Rumpty",
		Long: `Sign in and store a session for other rumpty commands.

By default, rumpty opens your browser to sign in.

To sign in with an API key from the Rumpty dashboard, use --token-stdin. It
asks for the key without showing it, or reads it from a pipe in scripts, so
the key stays out of your shell history and the process list.`,
		Example: `  rumpty login
  rumpty login --token-stdin
  printenv RUMPTY_API_KEY | rumpty login --token-stdin`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenStdin {
				if apiKey != "" {
					return config.NewUsageError("use either --token or --token-stdin, not both")
				}
				key, err := readToken(rt)
				if err != nil {
					return err
				}
				apiKey = key
			}
			return auth.Login(cmd.Context(), rt, apiKey, auth.LoginOptions{NoBrowser: noBrowser})
		},
	}

	cmd.Flags().StringVar(&apiKey, "token", "", "API key from the Rumpty dashboard (visible in shell history; prefer --token-stdin)")
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "Read the API key from stdin, or ask for it without echo in a terminal")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser; use the printed URL only")

	return cmd
}

func readToken(rt *app.Runtime) (string, error) {
	in := rt.Streams.In
	if in == nil {
		in = os.Stdin
	}
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec // stdin fd is small on supported platforms
		return rumptyterm.Password(rt.Streams.ErrOut, "API key")
	}
	data, err := io.ReadAll(io.LimitReader(in, 64<<10))
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", errors.New("no API key on stdin")
	}
	return key, nil
}
