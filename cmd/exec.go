package commands

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func newExecCmd(rt *app.Runtime) *cobra.Command {
	var guestUser string
	var identityFile string
	var sshDebug bool
	var allocateTTY bool

	cmd := &cobra.Command{
		Use:   "exec <vm> -- <command>",
		Short: "Run a non-interactive command on a VM",
		Long: `Run a command on a workspace VM. Put the command after "--", or after the vm name.

Requires rumpty login, or $RUMPTY_API_KEY and a workspace ($RUMPTY_WORKSPACE or --ws).`,
		Example: `  rumpty exec my-vm -- uptime
  rumpty exec my-vm --ws acme -- bash -lc 'cd /app && git pull'
  rumpty exec my-vm -t -- sudo systemctl reload nginx`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			vmRef, command, err := ssh.ParseExecArgs(args)
			if err != nil {
				return config.NewUsageError("%v", err)
			}
			err = ssh.Exec(cmd.Context(), rt, vmRef, command, &ssh.Options{
				GuestUser:    guestUser,
				IdentityFile: identityFile,
				Debug:        sshDebug,
				AllocateTTY:  allocateTTY,
			})
			return exitSSHIfNeeded(err)
		},
	}

	cmd.Flags().StringVar(&guestUser, "user", "", "Guest username on the VM")
	cmd.Flags().StringVarP(&identityFile, "identity", "i", "", "Private key for VM login")
	cmd.Flags().BoolVar(&sshDebug, "ssh-debug", false, "Enable verbose OpenSSH debugging")
	cmd.Flags().BoolVarP(&allocateTTY, "tty", "t", false, "Allocate a pseudo-TTY for the remote command")

	return cmd
}

func exitSSHIfNeeded(err error) error {
	var exit *ssh.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.Code)
	}
	return err
}
