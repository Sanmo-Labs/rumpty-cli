package commands

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/volume"
)

func newVolumeCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage volumes",
		Long:  "Create and delete block-storage volumes. Attach them to a VM with `rumpty volume attach`.",
		Example: `  rumpty volume create data --size 20
  rumpty volume rm data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty volume")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newVolumeCreateCmd(rt))
	cmd.AddCommand(newVolumeRmCmd(rt))
	return cmd
}

func newVolumeCreateCmd(rt *app.Runtime) *cobra.Command {
	var size, zone, filesystem, description string
	cmd := &cobra.Command{
		Use:   "create <name> --size <gib>",
		Short: "Create a volume and wait until it is available",
		Example: `  rumpty volume create data --size 20
  rumpty volume create data --size 50 --zone olas-closet --filesystem xfs`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			gib, err := strconv.Atoi(strings.TrimSpace(size))
			if err != nil {
				return config.NewUsageError("--size must be a whole number of GiB")
			}
			return volume.Create(cmd.Context(), rt, args[0], gib, zone, filesystem, description)
		},
	}
	cmd.Flags().StringVar(&size, "size", "", "Size of the volume in GiB (1-1024)")
	_ = cmd.MarkFlagRequired("size")
	cmd.Flags().StringVar(&zone, "zone", "", "Zone for the volume (default: olas-closet)")
	cmd.Flags().StringVar(&filesystem, "filesystem", "", "Filesystem to format with: ext4 or xfs (default: ext4)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description for the volume")
	return cmd
}

func newVolumeRmCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm <name|uid>",
		Short:   "Delete a volume",
		Long:    "Delete a volume and everything stored on it. Detach the volume first; it cannot be deleted while attached.",
		Example: `  rumpty volume rm data`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return volume.Rm(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}
