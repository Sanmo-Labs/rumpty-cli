package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func newSSHCmd(rt *app.Runtime) *cobra.Command {
	var guestUser string
	var identityFile string
	var sshDebug bool

	cmd := &cobra.Command{
		Use:   "ssh <vm>",
		Short: "Open an SSH session to a workspace VM",
		Long: `Open an interactive shell in your Virtual Machine.

Requires rumpty login, or $RUMPTY_API_KEY and a workspace ($RUMPTY_WORKSPACE or --ws).`,
		Example: `  rumpty login
  rumpty ssh my-vm --ws acme-dev`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return exitSSHIfNeeded(ssh.Open(cmd.Context(), rt, args[0], &ssh.Options{
				GuestUser:    guestUser,
				IdentityFile: identityFile,
				Debug:        sshDebug,
			}))
		},
	}

	cmd.Flags().StringVar(&guestUser, "user", "", "Guest username on the VM")
	cmd.Flags().StringVarP(&identityFile, "identity", "i", "", "Private key for VM login")
	cmd.Flags().BoolVar(&sshDebug, "ssh-debug", false, "Enable verbose OpenSSH debugging")

	return cmd
}
