package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/auth"
)

func newLogoutCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the local Rumpty session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Logout(cmd.Context(), rt)
		},
	}
}
