package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
	"github.com/Sanmo-Labs/rumpty-cli/internal/version"
)

func NewRoot(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "rumpty",
		Short:         "CLI for the Rumpty cloud platform",
		Long:          "Manage workspaces, VMs, and resources on the Rumpty cloud platform from your terminal.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
	}

	cmd.SetVersionTemplate("rumpty {{.Version}}\n")

	cmd.PersistentFlags().StringVar(
		&rt.Config.APIURL,
		"api-url",
		"",
		fmt.Sprintf("Rumpty API base URL; $%s", config.EnvAPIURL),
	)
	cmd.PersistentFlags().StringVar(
		&rt.Config.Token,
		"token",
		"",
		fmt.Sprintf("Rumpty API key; $%s", config.EnvToken),
	)
	cmd.PersistentFlags().StringVar(
		&rt.Config.Workspace,
		"ws",
		"",
		fmt.Sprintf("Workspace slug; $%s", config.EnvWorkspace),
	)
	cmd.PersistentFlags().StringVar(
		&rt.Config.Workspace,
		"workspace",
		"",
		fmt.Sprintf("Workspace slug; alias for --ws, $%s", config.EnvWorkspace),
	)
	cmd.PersistentFlags().StringVar(
		&rt.Config.LogLevel,
		"log-level",
		"",
		fmt.Sprintf("Log level: error, warn, info, debug; $%s", rumptylog.EnvLevel),
	)
	cmd.PersistentFlags().BoolVarP(
		&rt.Config.Verbose,
		"verbose",
		"v",
		false,
		"Enable debug logging (same as --log-level=debug)",
	)

	cobra.OnInitialize(func() {
		rt.Config.Resolve()
		if err := rumptylog.Configure(rt.Config.LogLevelValue(), rt.Streams.ErrOut); err != nil {
			fmt.Fprintf(os.Stderr, "rumpty: %v\n", err)
			os.Exit(2)
		}
		rumptylog.Debug("rumpty CLI starting", "version", version.String(), "api_url", rt.Config.APIURL)
	})

	cmd.AddCommand(
		newLoginCmd(rt),
		newLogoutCmd(rt),
		newWorkspacesCmd(rt),
		newVMsCmd(rt),
		newSSHCmd(rt),
		newExecCmd(rt),
		newCopyCmd(rt),
	)

	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if !c.HasParent() {
			term.PrintBanner(c.OutOrStdout())
			fmt.Fprintf(c.OutOrStdout(), "rumpty %s\n", version.String())
		}
		defaultHelp(c, args)
	})

	return cmd
}
