package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func addOutputFlag(cmd *cobra.Command, value *string) {
	cmd.Flags().StringVarP(value, "output", "o", string(term.OutputTable), "Output format: table or json")
	_ = cmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(
		[]string{string(term.OutputTable), string(term.OutputJSON)}, cobra.ShellCompDirectiveNoFileComp))
}

func outputFormat(value string) (term.OutputFormat, error) {
	format, err := term.ParseOutputFormat(value)
	if err != nil {
		return "", config.NewUsageError("%v", err)
	}
	return format, nil
}
