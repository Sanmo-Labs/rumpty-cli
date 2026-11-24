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
	cmd.AddCommand(newWSInviteCmd(rt))
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

func newWSInviteCmd(rt *app.Runtime) *cobra.Command {
	var role string
	cmd := &cobra.Command{
		Use:   "invite <workspace> <email>",
		Short: "Invite someone to a workspace",
		Long:  "Email an invitation to join a workspace. The email contains a link with a secret token that never comes back through the API. Invitations expire after 72 hours.",
		Example: `  rumpty ws invite prod-team ada@example.com
  rumpty ws invite prod-team ada@example.com --role viewer`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return workspace.Invite(cmd.Context(), rt, args[0], args[1], role)
		},
	}
	cmd.Flags().StringVar(&role, "role", "developer", "Role the invitee gets when they accept: admin, developer or viewer")
	return cmd
}
