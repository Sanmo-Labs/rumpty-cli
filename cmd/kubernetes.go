package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/kubernetes"
)

func newK8sCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Manage Kubernetes clusters",
		Long:  "Manage Kubernetes clusters. Create them and fetch kubeconfigs with `rumpty k8s create` and `rumpty k8s kubeconfig`.",
		Example: `  rumpty k8s create staging
  rumpty k8s rm staging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty k8s")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newK8sCreateCmd(rt))
	cmd.AddCommand(newK8sRmCmd(rt))
	cmd.AddCommand(newK8sKubeconfigCmd(rt))
	return cmd
}

func newK8sRmCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <name|uid>",
		Short: "Delete a cluster and all of its nodes",
		Long:  "Delete a Kubernetes cluster. The cluster and all of its nodes are destroyed. This cannot be undone.",
		Example: `  rumpty k8s create staging
  rumpty k8s rm staging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return kubernetes.Rm(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}

func newK8sKubeconfigCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "kubeconfig <name|uid> [--output path]",
		Short: "Write a cluster's kubeconfig for kubectl",
		Long:  "Write a cluster's kubeconfig to a file so kubectl can use it. The default file is <cluster>-kubeconfig.yaml in the current directory, written with 0600 permissions. Viewers cannot read a cluster's kubeconfig.",
		Example: `  rumpty k8s kubeconfig staging
  rumpty k8s kubeconfig staging --output ~/.kube/rumpty-staging.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return kubernetes.Kubeconfig(cmd.Context(), rt, args[0], output)
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "Where to write the kubeconfig (default <cluster>-kubeconfig.yaml)")
	return cmd
}

func newK8sCreateCmd(rt *app.Runtime) *cobra.Command {
	var opts kubernetes.CreateOpts
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Kubernetes cluster",
		Long:  "Create a Kubernetes cluster and wait until it is running. Every field except the name is optional; the cluster runs in the default zone with one worker unless you set more.",
		Example: `  rumpty k8s create staging
  rumpty k8s create prod --workers 3 --control-plane-plan scale --worker-plan launch`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return kubernetes.Create(cmd.Context(), rt, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.Version, "version", "", "k3s version to install (default v1.36.2+k3s1)")
	cmd.Flags().StringVar(&opts.Image, "image", "", "Node operating system image (default ubuntu-2404-lts)")
	cmd.Flags().StringVar(&opts.ControlPlanePlanSlug, "control-plane-plan", "", "Compute plan for the control-plane node (default scale)")
	cmd.Flags().StringVar(&opts.WorkerPlanSlug, "worker-plan", "", "Compute plan for the worker nodes (default launch)")
	cmd.Flags().IntVar(&opts.Workers, "workers", 0, "Number of worker nodes (default 1, up to 3)")
	return cmd
}
