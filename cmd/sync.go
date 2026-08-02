package commands

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/auth"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/sync"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func newSyncCmd(rt *app.Runtime) *cobra.Command {
	var (
		deleteExtra bool
		dryRun      bool
		restore     bool
		public      bool
		watch       bool
		daemon      bool
		daemonRun   bool
		include     []string
		exclude     []string
	)

	cmd := &cobra.Command{
		Use:   "sync <local-path> [<bucket>[/prefix]]",
		Short: "Sync a local directory or file to Rumpty object storage",
		Long: `Sync a local directory or a single file to a bucket in Rumpty object storage.

When no bucket is given, one named after the directory is used. The bucket is
created on first sync, private by default (pass --public to make its files
publicly readable instead). Only new and changed files are transferred.
Use --restore to copy files from the bucket back to the
local directory.

Repeatable --include and --exclude glob patterns narrow which files sync.
Patterns without a slash match file names at any depth ("*.jpg"), and a
trailing /** matches a whole directory ("node_modules/**").

With --watch the command keeps running after the initial sync and pushes new
changes as they happen, until interrupted with Ctrl-C. --daemon does the same
but in a detached background process that survives closing the terminal;
inspect it with "rumpty sync status" and end it with "rumpty sync stop".`,
		Example: `  rumpty sync . --ws my-team
  rumpty sync ./photos my-backups
  rumpty sync ~/Documents/photos my-backups
  rumpty sync /var/log/app my-backups/logs
  rumpty sync ~/Desktop/report.pdf my-backups
  rumpty sync . my-backups --include "*.jpg,*.png"
  rumpty sync . my-backups --exclude "node_modules/**" --exclude "*.log"
  rumpty sync ./projects/app my-backups/app --dry-run
  rumpty sync ./photos my-backups --delete
  rumpty sync ./photos my-backups --watch
  rumpty sync ./photos my-backups --daemon
  rumpty sync status
  rumpty sync stop ./photos
  rumpty sync ~/restored-photos my-backups --restore`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deleteExtra && restore {
				return config.NewUsageError("--delete cannot be combined with --restore")
			}
			continuous := watch || daemon || daemonRun
			if continuous && restore {
				return config.NewUsageError("--watch/--daemon cannot be combined with --restore")
			}
			if continuous && dryRun {
				return config.NewUsageError("--watch/--daemon cannot be combined with --dry-run")
			}

			if strings.TrimSpace(rt.Config.Token) == "" {
				if !term.IsInteractive(rt.Streams.ErrOut) {
					return config.NewUsageError("missing rumpty login or $%s", config.EnvToken)
				}
				if err := auth.Login(cmd.Context(), rt, "", auth.LoginOptions{}); err != nil {
					return err
				}
				rt.Config.Resolve()
			}
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}

			bucketArg := ""
			if len(args) == 2 {
				bucketArg = args[1]
			}
			bucketName, prefix := splitBucketArg(bucketArg)
			if bucketName == "" {
				bucketName = defaultBucketName(args[0])
			}
			if bucketName == "" {
				return config.NewUsageError("could not derive a bucket name from %q; pass one explicitly", args[0])
			}

			opts := sync.Options{
				LocalPath:  args[0],
				BucketName: bucketName,
				Prefix:     prefix,
				Delete:     deleteExtra,
				DryRun:     dryRun,
				Restore:    restore,
				Public:     public,
				Include:    cleanPatterns(include),
				Exclude:    cleanPatterns(exclude),
			}
			switch {
			case daemonRun:
				ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
				defer stop()
				return sync.RunDaemon(ctx, rt, &opts)
			case daemon:
				return sync.SpawnDaemon(rt, &opts)
			case watch:
				ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
				defer stop()
				return sync.Watch(ctx, rt, &opts)
			default:
				return sync.Run(cmd.Context(), rt, &opts)
			}
		},
	}

	cmd.Flags().BoolVar(&deleteExtra, "delete", false, "Delete remote files that no longer exist locally")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be transferred without doing it")
	cmd.Flags().BoolVar(&restore, "restore", false, "Copy files from the bucket back to the local directory")
	cmd.Flags().BoolVar(&watch, "watch", false, "Keep running and sync changes as they happen")
	cmd.Flags().BoolVar(&daemon, "daemon", false, "Run --watch as a detached background process")
	cmd.Flags().BoolVar(&daemonRun, "daemon-run", false, "")
	_ = cmd.Flags().MarkHidden("daemon-run")
	cmd.Flags().BoolVar(&public, "public", false, "Create the bucket with publicly readable files (default is private)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Only sync files matching these globs (repeatable or comma-separated)")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Skip files matching these globs (repeatable or comma-separated)")

	cmd.AddCommand(newSyncStatusCmd(rt), newSyncStopCmd(rt))
	return cmd
}

func newSyncStatusCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show running sync daemons",
		Example: `  rumpty sync status`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sync.StatusDaemons(rt)
		},
	}
}

func newSyncStopCmd(rt *app.Runtime) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "stop [<path>|<bucket>]",
		Short: "Stop a running sync daemon",
		Example: `  rumpty sync stop ./photos
  rumpty sync stop my-backups
  rumpty sync stop --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return sync.StopDaemons(rt, target, all)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Stop every running sync daemon")
	return cmd
}

func cleanPatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitBucketArg(arg string) (string, string) {
	arg = strings.Trim(strings.TrimSpace(arg), "/")
	name, prefix, _ := strings.Cut(arg, "/")
	return name, strings.Trim(prefix, "/")
}

func defaultBucketName(localPath string) string {
	abs, err := filepath.Abs(localPath)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	base := filepath.Base(abs)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}
