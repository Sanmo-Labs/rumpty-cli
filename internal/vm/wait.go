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
	lifecyclePollInterval = 2 * time.Second
	lifecyclePollTimeout  = 10 * time.Minute
)

func WaitForLifecycle(ctx context.Context, rt *app.Runtime, target *api.VM, action, operationID string) error {
	if lifecycleReached(ctx, rt, target.Slug, action) {
		return nil
	}

	spin := term.StartSpinner(out(rt), waitLabel(action, target.Slug))
	defer spin.Stop()

	deadline := time.Now().Add(lifecyclePollTimeout)
	ticker := time.NewTicker(lifecyclePollInterval)
	defer ticker.Stop()

	workspace := strings.TrimSpace(rt.Config.Workspace)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s on %s", action, target.Slug)
		}

		if opID := strings.TrimSpace(operationID); opID != "" {
			op, err := rt.API().GetOperation(ctx, workspace, opID)
			if err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(op.Status)) {
			case "failed":
				return fmt.Errorf("%s failed for %s", action, target.Slug)
			case "succeeded":
				if lifecycleReached(ctx, rt, target.Slug, action) {
					return nil
				}
			}
		} else if lifecycleReached(ctx, rt, target.Slug, action) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func lifecycleReached(ctx context.Context, rt *app.Runtime, slug, action string) bool {
	switch action {
	case "delete":
		_, err := Find(ctx, rt, slug)
		return err != nil && strings.Contains(err.Error(), "not found")
	default:
		vm, err := Find(ctx, rt, slug)
		if err != nil {
			return false
		}
		return vmHasTargetStatus(action, &vm)
	}
}

func vmHasTargetStatus(action string, vm *api.VM) bool {
	status := vmDisplayStatus(vm)
	switch action {
	case "start", "reboot":
		return status == "running"
	case "stop":
		return status == "stopped"
	default:
		return false
	}
}

func vmDisplayStatus(vm *api.VM) string {
	if s := strings.TrimSpace(vm.DisplayStatus); s != "" {
		return s
	}
	return strings.TrimSpace(vm.Status)
}

func waitLabel(action, slug string) string {
	switch action {
	case "start":
		return fmt.Sprintf("Waiting for %s to start", slug)
	case "stop":
		return fmt.Sprintf("Waiting for %s to stop", slug)
	case "reboot":
		return fmt.Sprintf("Waiting for %s to reboot", slug)
	case "delete":
		return fmt.Sprintf("Waiting for %s to be deleted", slug)
	default:
		return fmt.Sprintf("Waiting for %s", slug)
	}
}

func doneLabel(action, slug string) string {
	switch action {
	case "start":
		return fmt.Sprintf("Started %s", slug)
	case "stop":
		return fmt.Sprintf("Stopped %s", slug)
	case "reboot":
		return fmt.Sprintf("Rebooted %s", slug)
	case "delete":
		return fmt.Sprintf("Deleted %s", slug)
	default:
		return fmt.Sprintf("Finished %s for %s", action, slug)
	}
}
