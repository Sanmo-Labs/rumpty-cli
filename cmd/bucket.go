package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/buckets"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
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
	cmd.AddCommand(newBucketKeysCmd(rt))
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

func newBucketKeysCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage a bucket's S3 access keys",
		Long:  "List, create and revoke the S3 access keys scoped to a bucket. A new key's secret is shown once at creation.",
		Example: `  rumpty bucket keys ls assets
  rumpty bucket keys create assets deploy --permissions read,list
  rumpty bucket keys revoke assets deploy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty bucket keys")
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newBucketKeysLsCmd(rt))
	cmd.AddCommand(newBucketKeysCreateCmd(rt))
	cmd.AddCommand(newBucketKeysRevokeCmd(rt))
	return cmd
}

func newBucketKeysLsCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "ls <bucket>",
		Short: "List a bucket's access keys",
		Example: `  rumpty bucket keys ls assets
  rumpty bucket keys ls assets -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			format, err := term.ParseOutputFormat(output)
			if err != nil {
				return config.NewUsageError("%v", err)
			}
			return buckets.Keys(cmd.Context(), rt, args[0], format)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output format: table or json")
	return cmd
}

func newBucketKeysCreateCmd(rt *app.Runtime) *cobra.Command {
	var permissions []string
	cmd := &cobra.Command{
		Use:   "create <bucket> <name>",
		Short: "Create an S3 access key for a bucket",
		Long:  "Create an S3 access key scoped to a bucket for use with any S3-compatible client. The secret_access_key is shown once.",
		Example: `  rumpty bucket keys create assets deploy
  rumpty bucket keys create assets deploy --permissions read,list`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return buckets.CreateKey(cmd.Context(), rt, args[0], args[1], permissions)
		},
	}
	cmd.Flags().StringSliceVar(&permissions, "permissions", nil, "Permissions for the key: read, write, list, delete (default all four)")
	return cmd
}

func newBucketKeysRevokeCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke <bucket> <key>",
		Short: "Revoke a bucket's access key",
		Example: `  rumpty bucket keys revoke assets deploy
  rumpty bucket keys revoke assets deploy --yes`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return buckets.RevokeKey(cmd.Context(), rt, args[0], args[1], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}
