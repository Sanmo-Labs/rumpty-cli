package vm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
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

	term.Statusf(out(rt), "Resolving VM %s", ref)
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	appName := strings.TrimSpace(name)
	if appName == "" {
		appName = fmt.Sprintf("port-%d", port)
	}
	term.Statusf(out(rt), "Requesting public route for %s:%d (%s)", target.Slug, port, protocol)
	result, err := rt.API().ExposeVMApp(ctx, workspace, target.UID, api.ExposeVMAppRequest{
		Name:     strings.TrimSpace(name),
		Port:     port,
		Protocol: protocol,
	}, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		term.Statusf(out(rt), "Waiting for %s to become reachable", appName)
		wait := operationWait{
			spinner: fmt.Sprintf("Exposing %s on %s", appName, target.Slug),
			timeout: fmt.Errorf("timed out waiting for %s to become reachable on %s", appName, target.Slug),
			failed:  fmt.Errorf("expose failed for %s on %s", appName, target.Slug),
		}
		if err := waitForOperation(ctx, rt, result.OperationID, wait); err != nil {
			return err
		}
	}

	fmt.Fprintf(rt.Streams.Out, "Exposed %s on %s\n", appDisplayName(&result.App), target.Slug)
	fmt.Fprintf(rt.Streams.Out, "URL: %s\n", result.App.URL)
	fmt.Fprintf(rt.Streams.Out, "View: rumpty vm expose ls %s --ws %s\n", target.Slug, workspace)
	fmt.Fprintf(rt.Streams.Out, "Remove: rumpty unexpose %s --ws %s --name %s\n", target.Slug, workspace, appDisplayName(&result.App))
	return nil
}

func Unexpose(ctx context.Context, rt *app.Runtime, ref string, name string) error {
	appName := strings.TrimSpace(name)
	if appName == "" {
		return fmt.Errorf("app name is required")
	}

	term.Statusf(out(rt), "Resolving VM %s", ref)
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	term.Statusf(out(rt), "Removing public route %s from %s", appName, target.Slug)
	result, err := rt.API().UnexposeVMApp(ctx, workspace, target.UID, appName, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		term.Statusf(out(rt), "Waiting for %s to be removed", appName)
		wait := operationWait{
			spinner: fmt.Sprintf("Removing %s from %s", appName, target.Slug),
			timeout: fmt.Errorf("timed out waiting for %s to be removed from %s", appName, target.Slug),
			failed:  fmt.Errorf("failed to remove %s from %s", appName, target.Slug),
		}
		if err := waitForOperation(ctx, rt, result.OperationID, wait); err != nil {
			return err
		}
	}

	fmt.Fprintf(rt.Streams.Out, "Unexposed %s on %s\n", appDisplayName(&result.App), target.Slug)
	return nil
}

type operationWait struct {
	spinner string
	timeout error
	failed  error
}

func waitForOperation(ctx context.Context, rt *app.Runtime, operationID string, wait operationWait) error {
	spin := term.StartSpinner(out(rt), wait.spinner)
	defer spin.Stop()

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
