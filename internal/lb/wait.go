package lb

import (
	"context"
	"errors"
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
	pollTimeout  = 10 * time.Minute
)

var (
	errOperationFailed  = errors.New("operation failed")
	errOperationTimeout = errors.New("timed out")
)

// waitForChange waits for a queued change to finish and returns the load
// balancer as it is afterwards.
func waitForChange(ctx context.Context, rt *app.Runtime, result *api.LoadBalancerOperationResult, current *api.LoadBalancer, action, label string, spin *term.Spinner) (api.LoadBalancer, error) {
	if result.LoadBalancer != nil {
		current = result.LoadBalancer
	}
	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		spin.Update(fmt.Sprintf("Waiting for %s %s", lbName(current), label))
		if err := waitForOperation(ctx, rt, result.OperationID); err != nil {
			return api.LoadBalancer{}, changeError(ctx, rt, current, action, err)
		}
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	lb, err := rt.API().GetLoadBalancer(ctx, workspace, current.UID)
	if err != nil {
		return api.LoadBalancer{}, err
	}
	if strings.EqualFold(lb.Status, "failed") {
		return api.LoadBalancer{}, failureError(&lb, action)
	}
	return lb, nil
}

func waitForOperation(ctx context.Context, rt *app.Runtime, operationID string) error {
	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	workspace := strings.TrimSpace(rt.Config.Workspace)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return errOperationTimeout
		}

		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "succeeded":
			return nil
		case "failed":
			return errOperationFailed
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// changeError explains a failed or stalled operation, using the load
// balancer's last_failure when the server recorded one.
func changeError(ctx context.Context, rt *app.Runtime, lb *api.LoadBalancer, action string, err error) error {
	switch {
	case errors.Is(err, errOperationTimeout) && action == "delete":
		return fmt.Errorf("delete of %s was accepted and billing has stopped, but routing is still being removed; check with: rumpty lb ls", lbName(lb))
	case errors.Is(err, errOperationTimeout):
		return fmt.Errorf("timed out waiting for %s on %s; check progress with: rumpty lb get %s", action, lbName(lb), lbName(lb))
	case errors.Is(err, errOperationFailed):
		workspace := strings.TrimSpace(rt.Config.Workspace)
		if latest, getErr := rt.API().GetLoadBalancer(ctx, workspace, lb.UID); getErr == nil {
			return failureError(&latest, action)
		}
		return failureError(lb, action)
	default:
		return err
	}
}

func failureError(lb *api.LoadBalancer, action string) error {
	msg := fmt.Sprintf("%s failed for %s", action, lbName(lb))
	if reason := strings.TrimSpace(lb.LastFailure); reason != "" {
		msg += ": " + reason
	}
	if action == "create" {
		msg += fmt.Sprintf("\nThe load balancer still exists and is billed. Fix it with: rumpty lb update %s ...\nOr remove it with: rumpty lb delete %s", lbName(lb), lbName(lb))
	}
	return errors.New(msg)
}

func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func out(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
