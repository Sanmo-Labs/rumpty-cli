package commands

import (
	"errors"
	"os"

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
		Long: `Request a short-lived SSH certificate from the Rumpty API and connect with ssh.

Run rumpty login first, or set $RUMPTY_API_KEY for CI and scripts.`,
		Example: `  rumpty login
  rumpty ssh my-vm --ws acme-dev`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			err := ssh.Open(cmd.Context(), rt, args[0], ssh.Options{
				GuestUser:    guestUser,
				IdentityFile: identityFile,
				Debug:        sshDebug,
			})
			var exit *ssh.ExitError
			if errors.As(err, &exit) {
				os.Exit(exit.Code)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&guestUser, "user", "", "Guest username on the VM")
	cmd.Flags().StringVarP(&identityFile, "identity", "i", "", "Private key for VM login")
	cmd.Flags().BoolVar(&sshDebug, "ssh-debug", false, "Enable verbose OpenSSH debugging")

	return cmd
}
