package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func Create(ctx context.Context, rt *app.Runtime, req *api.CreateVMRequest, wait bool) error {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().CreateVM(ctx, workspace, req, api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	if result.OperationID == "" || result.VMUID == "" {
		return fmt.Errorf("create VM %q: response did not include the VM and operation IDs", req.Name)
	}
	if result.Status == "failed" {
		return fmt.Errorf("create failed for %s (operation %s)", req.Name, result.OperationID)
	}

	w := rt.Streams.Out
	if result.IsReplay {
		fmt.Fprintf(w, "VM creation already requested for %s (VM %s, operation %s)\n", req.Name, result.VMUID, result.OperationID)
	} else {
		fmt.Fprintf(w, "Queued VM creation for %s (VM %s, operation %s)\n", req.Name, result.VMUID, result.OperationID)
	}
	if !wait {
		return nil
	}

	spin := term.StartSpinner(out(rt), waitLabel("create", req.Name))
	defer spin.Stop()
	target := api.VM{UID: result.VMUID, Name: req.Name}
	if err := WaitForLifecycle(ctx, rt, &target, "create", result.OperationID, spin); err != nil {
		return err
	}
	spin.Stop()
	fmt.Fprintln(w, term.Success(w, fmt.Sprintf("%s is running", req.Name)))
	return nil
}
