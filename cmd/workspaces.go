package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/workspace"
)

func newWorkspacesCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"ws", "workspace"},
		Short:   "List workspaces you can access",
		Long:    "List Rumpty workspaces for the authenticated user.",
		Example: `  rumpty login
  rumpty workspaces
  rumpty ws
  rumpty ws -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return workspace.List(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	cmd.AddCommand(newWSInvitationsCmd(rt))
	return cmd
}

func newWSInvitationsCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "invitations [workspace]",
		Short: "List a workspace's pending invitations",
		Long:  "List the pending invitations of a workspace. Accepted, revoked and expired invitations are not listed.",
		Example: `  rumpty ws invitations prod-team
  rumpty ws invitations prod-team -o json
  rumpty ws invitations revoke prod-team ada@example.com`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			wsRef := ""
			if len(args) == 1 {
				wsRef = args[0]
			}
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			return workspace.ListInvitations(cmd.Context(), rt, wsRef, format)
		},
	}
	addOutputFlag(cmd, &output)

	revoke := &cobra.Command{
		Use:   "revoke <workspace> <invitation>",
		Short: "Revoke a pending invitation",
		Long:  "Revoke a pending workspace invitation by uid or email. Its link stops working straight away.",
		Example: `  rumpty ws invitations revoke prod-team 019a8f30-4c5d-7e6f-8a9b-0c1d2e3f4a5b
  rumpty ws invitations revoke prod-team ada@example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return workspace.RevokeInvitation(cmd.Context(), rt, args[0], args[1])
		},
	}
	cmd.AddCommand(revoke)
	return cmd
}
