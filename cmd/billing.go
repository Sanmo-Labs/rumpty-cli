package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/billing"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

func newBillingCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "View billing for your account",
		Long:  "View invoices and billing information for your Rumpty account.",
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
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return billing.List(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	cmd.AddCommand(newBillingInvoiceDownloadCmd(rt))
	return cmd
}

func newBillingInvoiceDownloadCmd(rt *app.Runtime) *cobra.Command {
	var output string
	var force bool
	cmd := &cobra.Command{
		Use:   "download <id>",
		Short: "Download an invoice PDF",
		Example: `  rumpty billing invoices download 019a4c1f-8d9e-7f0a-8b1c-2d3e4f5a6b7c
  rumpty billing invoices download 019a4c1f-8d9e-7f0a-8b1c-2d3e4f5a6b7c --output invoice.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return billing.Download(cmd.Context(), rt, args[0], output, force)
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "Path to write the PDF to (default: the filename the server suggests)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite the output file if it exists")
	return cmd
}
