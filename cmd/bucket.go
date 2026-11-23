package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/buckets"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

func newBucketCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage objects in a bucket",
		Long:  "Download, list and delete the objects stored in a Rumpty bucket.",
		Example: `  rumpty bucket objects assets
  rumpty bucket objects assets reports/
  rumpty bucket rm-object assets/reports/2026/report.pdf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty bucket")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newBucketObjectsCmd(rt))
	cmd.AddCommand(newBucketCpCmd(rt))
	cmd.AddCommand(newBucketRmObjectCmd(rt))
	return cmd
}

func newBucketObjectsCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "objects <bucket> [prefix]",
		Short: "List the objects in a bucket",
		Example: `  rumpty bucket objects assets
  rumpty bucket objects assets reports/2026/
  rumpty bucket objects assets -o json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			prefix := ""
			if len(args) == 2 {
				prefix = args[1]
			}
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			return buckets.Objects(cmd.Context(), rt, args[0], prefix, format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}

func newBucketRmObjectCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm-object <bucket>/<key>",
		Short:   "Delete an object from a bucket",
		Long:    "Delete an object from a bucket. The deletion is immediate and cannot be undone.",
		Example: `  rumpty bucket rm-object assets/reports/2026/report.pdf`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return buckets.RmObject(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}

func newBucketCpCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cp <bucket>/<key> <local>",
		Short:   "Download one object from a bucket",
		Long:    "Download one object from a bucket to a local file. In a private bucket the download goes through a presigned read URL.",
		Example: `  rumpty bucket cp assets/reports/2026/report.pdf ./report.pdf`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return buckets.Cp(cmd.Context(), rt, args[0], args[1])
		},
	}
	return cmd
}
