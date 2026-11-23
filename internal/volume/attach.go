package volume

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Attach attaches an available volume to a running VM in the same zone and
// waits for the operation to finish.
func Attach(ctx context.Context, rt *app.Runtime, volumeRef, vmRef string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding volume")
	volumes, err := rt.API().ListVolumes(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findVolume(volumes, volumeRef, workspace)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Finding VM")
	vms, err := rt.API().ListVMs(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	vm, err := findVM(vms, vmRef, workspace)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Attaching "+target.Name+" to "+vm.Name)
	result, err := rt.API().AttachVolume(ctx, workspace, target.UID, vm.UID)
	if err != nil {
		spin.Stop()
		return err
	}
	if err := waitForVolume(ctx, rt, workspace, result.OperationID, target.Name, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Attached volume %s to %s", target.Name, vm.Name)))
	return nil
}

// Detach detaches an in-use volume from its VM and waits for the operation
// to finish. Its data is kept.
func Detach(ctx context.Context, rt *app.Runtime, volumeRef string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding volume")
	volumes, err := rt.API().ListVolumes(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findVolume(volumes, volumeRef, workspace)
	if err != nil {
		return err
	}
	if target.VmUID == "" {
		return fmt.Errorf("volume %s is not attached to a VM", target.Name)
	}

	spin = term.StartLoading(w, "Detaching "+target.Name)
	result, err := rt.API().DetachVolume(ctx, workspace, target.UID)
	if err != nil {
		spin.Stop()
		return err
	}
	if err := waitForVolume(ctx, rt, workspace, result.OperationID, target.Name, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Detached volume %s", target.Name)))
	return nil
}

func findVM(vms []api.VM, ref, workspaceSlug string) (api.VM, error) {
	var matches []api.VM
	for i := range vms {
		v := &vms[i]
		if v.UID == ref || v.Name == ref || v.Slug == ref {
			matches = append(matches, *v)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.VM{}, fmt.Errorf("no VM %q in workspace %s", ref, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.VM{}, fmt.Errorf("multiple VMs match %q: %s", ref, strings.Join(names, ", "))
	}
}
