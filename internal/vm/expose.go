package vm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

const (
	exposePollInterval = 2 * time.Second
	exposePollTimeout  = 2 * time.Minute
)

func Expose(ctx context.Context, rt *app.Runtime, ref string, port int, name string, protocol string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if protocol == "" {
		protocol = "http"
	}

	spin := term.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	appName := strings.TrimSpace(name)
	if appName == "" {
		appName = fmt.Sprintf("port-%d", port)
	}
	spin.Update(fmt.Sprintf("Requesting a public URL for %s:%d (%s)", target.Slug, port, protocol))
	result, err := rt.API().ExposeVMApp(ctx, workspace, target.UID, api.ExposeVMAppRequest{
		Name:     strings.TrimSpace(name),
		Port:     port,
		Protocol: protocol,
	}, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		spin.Update(fmt.Sprintf("Waiting for %s to become reachable", appName))
		wait := operationWait{
			timeout: fmt.Errorf("timed out waiting for %s to become reachable on %s", appName, target.Slug),
			failed:  fmt.Errorf("expose failed for %s on %s", appName, target.Slug),
		}
		if err := waitForOperation(ctx, rt, result.OperationID, wait); err != nil {
			return err
		}
	}

	spin.Stop()
	w := rt.Streams.Out
	fmt.Fprintln(w, term.Success(w, fmt.Sprintf("Exposed %s on %s", appDisplayName(&result.App), target.Slug)))
	fmt.Fprintf(w, "URL: %s\n", term.Link(w, result.App.URL))
	fmt.Fprintln(w, term.Muted(w, fmt.Sprintf("View: rumpty vm expose ls %s --ws %s", target.Slug, workspace)))
	fmt.Fprintln(w, term.Muted(w, fmt.Sprintf("Remove: rumpty unexpose %s --ws %s --name %s", target.Slug, workspace, appDisplayName(&result.App))))
	return nil
}

func Unexpose(ctx context.Context, rt *app.Runtime, ref string, name string) error {
	appName := strings.TrimSpace(name)
	if appName == "" {
		return fmt.Errorf("app name is required")
	}

	spin := term.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin.Update(fmt.Sprintf("Removing %s from %s", appName, target.Slug))
	result, err := rt.API().UnexposeVMApp(ctx, workspace, target.UID, appName, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		spin.Update(fmt.Sprintf("Waiting for %s to be removed", appName))
		wait := operationWait{
			timeout: fmt.Errorf("timed out waiting for %s to be removed from %s", appName, target.Slug),
			failed:  fmt.Errorf("failed to remove %s from %s", appName, target.Slug),
		}
		if err := waitForOperation(ctx, rt, result.OperationID, wait); err != nil {
			return err
		}
	}

	spin.Stop()
	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Unexposed %s on %s", appDisplayName(&result.App), target.Slug)))
	return nil
}

type operationWait struct {
	timeout error
	failed  error
}

func waitForOperation(ctx context.Context, rt *app.Runtime, operationID string, wait operationWait) error {
	deadline := time.Now().Add(exposePollTimeout)
	ticker := time.NewTicker(exposePollInterval)
	defer ticker.Stop()

	workspace := strings.TrimSpace(rt.Config.Workspace)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return wait.timeout
		}

		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "succeeded":
			return nil
		case "failed":
			return wait.failed
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func appDisplayName(app *api.VMApp) string {
	if app == nil {
		return "service"
	}
	if s := strings.TrimSpace(app.Slug); s != "" {
		return s
	}
	if s := strings.TrimSpace(app.Name); s != "" {
		return s
	}
	return "service"
}
