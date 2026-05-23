package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func newCopyCmd(rt *app.Runtime) *cobra.Command {
	var guestUser string
	var identityFile string
	var sshDebug bool
	var recursive bool

	cmd := &cobra.Command{
		Use:     "copy <src> <dest>",
		Aliases: []string{"cp"},
		Short:   "Copy files between your machine and a VM",
		Long: `Copy files to or from a VM using rsync when available.

One path must be remote using vm:path syntax.
Falls back to scp when rsync is missing locally or on the VM.
Uploading a local directory uses recursive copy automatically; use -r when
downloading a remote directory via scp.

Requires rumpty login, or $RUMPTY_API_KEY and a workspace ($RUMPTY_WORKSPACE or --ws).`,
		Example: `  rumpty copy ./app.tar.gz my-vm:/tmp/
  rumpty cp my-vm:/var/log/app.log ./logs/
  rumpty cp ./dist my-vm:/srv/app/
  rumpty cp -r my-vm:/var/log ./logs/`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			err := ssh.Copy(cmd.Context(), rt, args[0], args[1], &ssh.Options{
				GuestUser:    guestUser,
				IdentityFile: identityFile,
				Debug:        sshDebug,
			}, recursive)
			return exitSSHIfNeeded(err)
		},
	}

	cmd.Flags().StringVar(&guestUser, "user", "", "Guest username on the VM")
	cmd.Flags().StringVarP(&identityFile, "identity", "i", "", "Private key for VM login")
	cmd.Flags().BoolVar(&sshDebug, "ssh-debug", false, "Enable verbose OpenSSH debugging")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Copy directories recursively (required for remote dirs when using scp)")

	return cmd
}
