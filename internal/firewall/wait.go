package firewall

import (
	"context"
	"fmt"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// waitForPolicyApply polls a firewall.apply operation until it succeeds or
// fails, so commands that change a policy report the real outcome.
func waitForPolicyApply(ctx context.Context, rt *app.Runtime, workspace, operationID string, spin *term.Spinner) error {
	deadline := time.After(90 * time.Second)
	for {
		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch op.Status {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("the operation failed (%s)", op.ResourceUID)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("the apply operation %s did not finish in 90s (operation %s, status %s)", operationID, operationID, op.Status)
		case <-time.After(2 * time.Second):
			spin.Update("Applying the policy")
		}
	}
}
