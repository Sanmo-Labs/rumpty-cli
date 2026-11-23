package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

const (
	pollInterval = 2 * time.Second
	pollTimeout  = 15 * time.Minute
)

// Rm deletes a cluster matched by uid, name or slug and waits for the delete
// operation to finish.
func Rm(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
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

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete cluster %s without confirmation; pass --yes to skip the prompt", target.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete cluster %s? The cluster and all of its nodes are destroyed. This cannot be undone", target.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Deleting "+target.Name)
	result, err := rt.API().DeleteCluster(ctx, workspace, target.UID)
	if err != nil {
		spin.Stop()
		return err
	}
	if err := waitForCluster(ctx, rt, workspace, result.OperationID, target.Name, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Deleted cluster %s", target.Name)))
	return nil
}

// waitForCluster polls the cluster operation until it succeeds or fails.
func waitForCluster(ctx context.Context, rt *app.Runtime, workspace, operationID, name string, spin *term.Spinner) error {
	if strings.TrimSpace(operationID) == "" {
		return nil
	}
	spin.Update("Waiting for " + name)

	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for cluster %s", name)
		}

		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "failed":
			return fmt.Errorf("operation failed for cluster %s", name)
		case "succeeded":
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func findCluster(clusters []api.Cluster, ref, workspaceSlug string) (api.Cluster, error) {
	var matches []api.Cluster
	for i := range clusters {
		c := &clusters[i]
		if c.UID == ref || c.Name == ref || c.Slug == ref {
			matches = append(matches, *c)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Cluster{}, fmt.Errorf("no cluster %q in workspace %s", ref, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.Cluster{}, fmt.Errorf("multiple clusters match %q: %s", ref, strings.Join(names, ", "))
	}
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
