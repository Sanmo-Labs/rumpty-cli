package db

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

const (
	deletePollInterval = 2 * time.Second
	deletePollTimeout  = 10 * time.Minute
)

// Delete queues deletion of a database, confirming with the user first unless
// yes is set. The database's snapshots are kept.
func Delete(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	started := time.Now()
	spin := term.StartSpinner(w, "Finding "+ref)
	defer spin.Stop()

	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete database %s without confirmation; pass --yes to skip the prompt", target.Slug)
		}
		spin.Stop()
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete database %s? All data is destroyed and snapshots are detached", target.Slug))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
		spin = term.StartSpinner(w, "Deleting "+target.Slug)
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().DeleteDatabase(ctx, workspace, target.UID, api.NewIdempotencyKey())
	if err != nil {
		return err
	}

	if err := waitForDelete(ctx, rt, workspace, result.OperationID, target.Slug, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(w, term.Success(w, fmt.Sprintf("Deleted %s", target.Slug)+term.Muted(w, " ("+formatElapsed(time.Since(started))+")")))
	return nil
}

func waitForDelete(ctx context.Context, rt *app.Runtime, workspace, operationID, slug string, spin *term.Spinner) error {
	spin.Update("Waiting for " + slug + " to be deleted")

	deadline := time.Now().Add(deletePollTimeout)
	ticker := time.NewTicker(deletePollInterval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to be deleted", slug)
		}

		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "failed":
			return fmt.Errorf("delete failed for %s", slug)
		case "succeeded":
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
