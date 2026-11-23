package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/auth"
)

func newLogoutCmd(rt *app.Runtime) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the local Rumpty session",
		Long: `Remove the local Rumpty session.

With --all, also end every other session on your account: other machines,
other CLI logins and the console. API keys stay valid; revoke them in the
console.`,
		Example: `  rumpty logout
  rumpty logout --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return auth.LogoutAll(cmd.Context(), rt)
			}
			return auth.Logout(cmd.Context(), rt)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Sign out of every session on your account, on all devices")
	return cmd
}
