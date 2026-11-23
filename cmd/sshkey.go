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
	cmd.AddCommand(newSSHKeyRmCmd(rt))
	return cmd
}

func newSSHKeyRmCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <name|uid|fingerprint>",
		Short: "Remove an SSH public key from your account",
		Example: `  rumpty ssh-key rm laptop
  rumpty ssh-key rm 019a7b10-2c3d-7e4f-8a9b-0c1d2e3f4a5b`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return sshkeys.Rm(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
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
