package volume

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
	pollInterval = 2 * time.Second
	pollTimeout  = 5 * time.Minute
)

// Create queues a new volume and waits until it is available.
func Create(ctx context.Context, rt *app.Runtime, name string, sizeGiB int, zone, filesystem, description string) error {
	if sizeGiB < 1 || sizeGiB > 1024 {
		return fmt.Errorf("volume size must be between 1 and 1024 GiB, got %d", sizeGiB)
	}

	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin := term.StartLoading(w, "Creating volume "+name)
	result, err := rt.API().CreateVolume(ctx, workspace, api.CreateVolumeRequest{
		Name:        name,
		Description: description,
		SizeGiB:     sizeGiB,
		ZoneSlug:    zone,
		Filesystem:  filesystem,
	})
	if err != nil {
		spin.Stop()
		return err
	}

	volume := result.Volume
	if err := waitForVolume(ctx, rt, workspace, result.OperationID, volume.Name, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Created volume %s (%d GiB, %s)", volume.Name, volume.SizeGiB, volume.ZoneSlug)))
	return nil
}

// Rm deletes a volume matched by uid, name or slug.
func Rm(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding volume")
	volumes, err := rt.API().ListVolumes(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findVolume(volumes, ref, workspace)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete volume %s without confirmation; pass --yes to skip the prompt", target.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete volume %s? Everything stored on it is destroyed. Detach it first", target.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Deleting "+target.Name)
	result, err := rt.API().DeleteVolume(ctx, workspace, target.UID)
	if err != nil {
		spin.Stop()
		return err
	}
	if err := waitForVolume(ctx, rt, workspace, result.OperationID, target.Name, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Deleted volume %s", target.Name)))
	return nil
}

// waitForVolume polls the volume operation until it succeeds or fails.
func waitForVolume(ctx context.Context, rt *app.Runtime, workspace, operationID, name string, spin *term.Spinner) error {
	if strings.TrimSpace(operationID) == "" {
		return nil
	}
	spin.Update("Waiting for " + name)

	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for volume %s", name)
		}

		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "failed":
			return fmt.Errorf("operation failed for volume %s", name)
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

func findVolume(volumes []api.Volume, ref, workspaceSlug string) (api.Volume, error) {
	var matches []api.Volume
	for i := range volumes {
		v := &volumes[i]
		if v.UID == ref || v.Name == ref || v.Slug == ref {
			matches = append(matches, *v)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Volume{}, fmt.Errorf("no volume %q in workspace %s", ref, workspaceSlug)
	default:
		uids := make([]string, 0, len(matches))
		for i := range matches {
			uids = append(uids, matches[i].UID)
		}
		return api.Volume{}, fmt.Errorf("multiple volumes match %q: %s", ref, strings.Join(uids, ", "))
	}
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
