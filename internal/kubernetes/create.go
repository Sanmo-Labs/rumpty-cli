package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// CreateOpts carries the optional flags for creating a cluster.
type CreateOpts struct {
	Version              string
	Image                string
	ControlPlanePlanSlug string
	WorkerPlanSlug       string
	Workers              int
}

// Create queues a new Kubernetes cluster and waits until it is running.
func Create(ctx context.Context, rt *app.Runtime, name string, opts CreateOpts) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Creating cluster "+name)
	result, err := rt.API().CreateCluster(ctx, workspace, &api.CreateClusterRequest{
		Name:                 name,
		Version:              strings.TrimSpace(opts.Version),
		ImageSlug:            strings.TrimSpace(opts.Image),
		ControlPlanePlanSlug: strings.TrimSpace(opts.ControlPlanePlanSlug),
		WorkerPlanSlug:       strings.TrimSpace(opts.WorkerPlanSlug),
		WorkerCount:          opts.Workers,
	}, api.NewIdempotencyKey())
	if err != nil {
		spin.Stop()
		return err
	}
	if err := waitForCluster(ctx, rt, workspace, result.OperationID, name, spin); err != nil {
		return err
	}

	// The create operation succeeded; wait for the cluster itself to read
	// running, since provisioning continues after the operation.
	spin.Update("Waiting for " + name + " to be ready")
	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for cluster %s to become ready", name)
		}
		cluster, err := rt.API().GetCluster(ctx, workspace, result.Cluster.UID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(cluster.Status)) {
		case "running":
			spin.Stop()
			fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Cluster %s is running (endpoint %s)", cluster.Name, cluster.Endpoint)))
			return nil
		case "failed":
			spin.Stop()
			return fmt.Errorf("cluster %s failed to provision", name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
