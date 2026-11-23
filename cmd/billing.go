package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/billing"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newBillingCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Show your invoices",
		Long:  "Show the invoices RumptyCloud has issued for completed billing cycles.",
		Example: `  rumpty billing invoices
  rumpty billing invoices -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty billing")
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newBillingInvoicesCmd(rt))
	return cmd
}

func newBillingInvoicesCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "invoices",
		Short: "List your invoices",
		Example: `  rumpty billing invoices
  rumpty billing invoices -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			format, err := term.ParseOutputFormat(output)
			if err != nil {
				return config.NewUsageError("%v", err)
			}
			return billing.Invoices(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}
