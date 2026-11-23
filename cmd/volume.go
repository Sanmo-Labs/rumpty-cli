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
	cmd.AddCommand(newVolumeAttachCmd(rt))
	cmd.AddCommand(newVolumeDetachCmd(rt))
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

func newVolumeAttachCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attach <volume> <vm>",
		Short:   "Attach a volume to a VM",
		Long:    "Attach an available volume to a running VM in the same zone. On its first attach the volume is formatted with its filesystem and mounted under /mnt/rumpty/.",
		Example: `  rumpty volume attach data web-1`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return volume.Attach(cmd.Context(), rt, args[0], args[1])
		},
	}
	return cmd
}

func newVolumeDetachCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "detach <volume>",
		Short:   "Detach a volume from its VM",
		Long:    "Detach an in-use volume from its VM. Its data is kept. Unmount the filesystem inside the VM first to avoid losing unwritten data.",
		Example: `  rumpty volume detach data`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return volume.Detach(cmd.Context(), rt, args[0])
		},
	}
	return cmd
}
