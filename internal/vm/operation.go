package vm

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
	w := out(rt)
	started := time.Now()
	spin := term.StartSpinner(w, "Finding "+ref)
	defer spin.Stop()

	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	if vmHasTargetStatus(action, &target) {
		spin.Stop()
		fmt.Fprintln(w, term.Muted(w, fmt.Sprintf("%s is already %s", target.Slug, vmDisplayStatus(&target))))
		return nil
	}

	spin.Update(actionLabel(action, target.Slug))
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := fn(ctx, workspace, target.UID, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if err := WaitForLifecycle(ctx, rt, &target, action, result.OperationID, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(w, term.Success(w, fmt.Sprintf("%s %s", doneLabel(action, target.Slug), term.Muted(w, "("+formatElapsed(time.Since(started))+")"))))
	return nil
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
