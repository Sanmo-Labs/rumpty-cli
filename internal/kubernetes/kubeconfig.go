package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	if err := writeKubeconfigFile(path, []byte(cluster.Kubeconfig)); err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Wrote the kubeconfig for %s to %s (use it with: kubectl --kubeconfig %s)", cluster.Name, path, path)))
	return nil
}

// writeKubeconfigFile writes the kubeconfig with owner-only permissions,
// replacing any existing file atomically so a previous file is never left
// with its old, wider permissions or half-written content.
func writeKubeconfigFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	////nolint:gosec // G703: the output path is caller-supplied by design; the file is written to a temp file in its own directory and renamed
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
