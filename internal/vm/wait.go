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
	lifecyclePollInterval = 2 * time.Second
	lifecyclePollTimeout  = 10 * time.Minute
)

// WaitForLifecycle polls until the action has taken effect, showing progress
// on spin.
func WaitForLifecycle(ctx context.Context, rt *app.Runtime, target *api.VM, action, operationID string, spin *term.Spinner) error {
	name := target.Slug
	if action == "create" {
		name = target.Name
	} else {
		if lifecycleReached(ctx, rt, target.Slug, action) {
			return nil
		}
		spin.Update(waitLabel(action, name))
	}

	deadline := time.Now().Add(lifecyclePollTimeout)
	ticker := time.NewTicker(lifecyclePollInterval)
	defer ticker.Stop()

	workspace := strings.TrimSpace(rt.Config.Workspace)
	opID := strings.TrimSpace(operationID)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s on %s", action, name)
		}

		checkVM := opID == ""
		if opID != "" {
			op, err := rt.API().GetOperation(ctx, workspace, opID)
			if err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(op.Status)) {
			case "failed":
				return fmt.Errorf("%s failed for %s", action, name)
			case "succeeded":
				checkVM = true
			}
		}
		if checkVM {
			if action == "create" {
				running, err := createdVMRunning(ctx, rt, workspace, target.UID)
				if err != nil || running {
					return err
				}
			} else if lifecycleReached(ctx, rt, target.Slug, action) {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func createdVMRunning(ctx context.Context, rt *app.Runtime, workspace, uid string) (bool, error) {
	vms, err := rt.API().ListVMs(ctx, workspace)
	if err != nil {
		return false, err
	}
	for i := range vms {
		if vms[i].UID != uid {
			continue
		}
		switch status := vmDisplayStatus(&vms[i]); status {
		case "running":
			return true, nil
		case "failed", "deleted", "deleting":
			return false, fmt.Errorf("VM %s is %s", uid, status)
		}
		break
	}
	return false, nil
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
	case "create", "start":
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

func actionLabel(action, slug string) string {
	switch action {
	case "start":
		return "Starting " + slug
	case "stop":
		return "Stopping " + slug
	case "reboot":
		return "Rebooting " + slug
	case "delete":
		return "Deleting " + slug
	default:
		return fmt.Sprintf("Running %s on %s", action, slug)
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
