package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Kubeconfig writes a cluster's kubeconfig so kubectl can use it. The
// default file is <cluster-slug>-kubeconfig.yaml in the current directory,
// and the file is written with 0600 permissions.
func Kubeconfig(ctx context.Context, rt *app.Runtime, ref, output string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding cluster")
	clusters, err := rt.API().ListClusters(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findCluster(clusters, ref, workspace)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Fetching the kubeconfig for "+target.Name)
	cluster, err := rt.API().GetCluster(ctx, workspace, target.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	if cluster.Kubeconfig == "" {
		if cluster.KubeconfigRestricted {
			return fmt.Errorf("your role cannot read the kubeconfig for %s; the kubeconfig needs the secrets read permission (owner, admin or developer)", cluster.Name)
		}
		return fmt.Errorf("the kubeconfig for %s is not ready yet; it is being fetched, try again in a couple of minutes", cluster.Name)
	}

	path := strings.TrimSpace(output)
	if path == "" {
		path = fmt.Sprintf("%s-kubeconfig.yaml", cluster.Slug)
	}
	if err := os.WriteFile(path, []byte(cluster.Kubeconfig), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Wrote the kubeconfig for %s to %s (use it with: kubectl --kubeconfig %s)", cluster.Name, path, path)))
	return nil
}
