package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/sshkeys"
)

func newSSHKeyCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage your SSH public keys",
		Long:  "Add and remove the SSH public keys on your Rumpty account.",
		Example: `  rumpty ssh-key add
  rumpty ssh-key add ~/.ssh/id_ed25519.pub --name laptop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty ssh-key")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSSHKeyAddCmd(rt))
	return cmd
}

func newSSHKeyAddCmd(rt *app.Runtime) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add [path]",
		Short: "Add an SSH public key to your account",
		Example: `  rumpty ssh-key add
  rumpty ssh-key add ~/.ssh/id_ed25519.pub
  rumpty ssh-key add ~/.ssh/id_ed25519.pub --name laptop`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return sshkeys.Add(cmd.Context(), rt, path, name)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Label for the key (default: the key's comment)")
	return cmd
}
