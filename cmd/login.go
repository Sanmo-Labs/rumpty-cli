package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/auth"
)

func newLoginCmd(rt *app.Runtime) *cobra.Command {
	var apiKey string
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Rumpty",
		Long: `Sign in and store a session for other rumpty commands.

By default, rumpty opens your browser to sign in.

For CI/CD or scripts, create an API key in the Rumpty dashboard, then run:

  rumpty login --token <api-key>`,
		Example: `  rumpty login
  rumpty login --token "$RUMPTY_API_KEY"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Login(cmd.Context(), rt, apiKey, auth.LoginOptions{NoBrowser: noBrowser})
		},
	}

	cmd.Flags().StringVar(&apiKey, "token", "", "API key from the Rumpty dashboard")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser; use the printed URL only")

	return cmd
}
