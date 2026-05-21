package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
)

type lifecycleFunc func(ctx context.Context, workspace, vmUID, idempotency string) (api.VMOperationResult, error)

func Start(ctx context.Context, rt *app.Runtime, ref string) error {
	return runLifecycle(ctx, rt, ref, "start", rt.API().StartVM)
}

func Stop(ctx context.Context, rt *app.Runtime, ref string) error {
	return runLifecycle(ctx, rt, ref, "stop", rt.API().StopVM)
}

func Reboot(ctx context.Context, rt *app.Runtime, ref string) error {
	return runLifecycle(ctx, rt, ref, "reboot", rt.API().RebootVM)
}

func Delete(ctx context.Context, rt *app.Runtime, ref string) error {
	return runLifecycle(ctx, rt, ref, "delete", rt.API().DeleteVM)
}

func runLifecycle(ctx context.Context, rt *app.Runtime, ref, action string, fn lifecycleFunc) error {
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	if vmHasTargetStatus(action, &target) {
		fmt.Fprintln(out(rt), doneLabel(action, target.Slug))
		return nil
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := fn(ctx, workspace, target.UID, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if err := WaitForLifecycle(ctx, rt, &target, action, result.OperationID); err != nil {
		return err
	}

	fmt.Fprintln(out(rt), doneLabel(action, target.Slug))
	return nil
}

func out(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
