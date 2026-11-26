package lb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	rumptyterm "github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// ErrNotConfirmed is returned when the user declines a delete prompt.
var ErrNotConfirmed = errors.New("aborted")

func Create(ctx context.Context, rt *app.Runtime, name string, s *Settings) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("load balancer name is required")
	}
	if s.Port == nil {
		return fmt.Errorf("port is required")
	}
	if len(s.VMs) == 0 {
		return fmt.Errorf("at least one VM is required")
	}
	if err := s.Validate(); err != nil {
		return err
	}

	started := time.Now()
	spin := rumptyterm.StartSpinner(out(rt), "Finding VMs")
	defer spin.Stop()
	vms, err := resolveVMs(ctx, rt, s.VMs)
	if err != nil {
		return err
	}

	req := api.CreateLoadBalancerRequest{
		Name:           name,
		TargetPort:     *s.Port,
		VMUIDs:         vmUIDs(vms),
		HealthCheck:    s.healthCheck(),
		StickySessions: s.Sticky,
	}
	if s.Protocol != nil {
		req.TargetProtocol = *s.Protocol
	}
	if s.AllowedCIDRs != nil {
		req.AllowedCIDRs = *s.AllowedCIDRs
	}

	spin.Update(fmt.Sprintf("Creating load balancer %s for %s", name, strings.Join(vmSlugs(vms), ", ")))
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().CreateLoadBalancer(ctx, workspace, &req, api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	if result.LoadBalancer == nil {
		return fmt.Errorf("create load balancer %s: response did not include the load balancer", name)
	}

	lb, err := waitForChange(ctx, rt, &result, result.LoadBalancer, "create", "to be ready", spin)
	if err != nil {
		return err
	}
	spin.Stop()

	w := rt.Streams.Out
	fmt.Fprintln(w, rumptyterm.Success(w, fmt.Sprintf("Created load balancer %s %s", lbName(&lb), rumptyterm.Muted(w, "("+formatElapsed(time.Since(started))+")"))))
	fmt.Fprintf(w, "URL: %s\n", rumptyterm.Link(w, lb.URL))
	fmt.Fprintf(w, "Targets: %s\n", targetNames(&lb))
	fmt.Fprintln(w, rumptyterm.Muted(w, "Billed $2/month, hourly, until you delete it."))
	fmt.Fprintln(w, rumptyterm.Muted(w, fmt.Sprintf("View: rumpty lb get %s --ws %s", lbName(&lb), workspace)))
	return nil
}

func Update(ctx context.Context, rt *app.Runtime, ref string, s *Settings) error {
	if s.IsEmpty() {
		return fmt.Errorf("nothing to update; pass at least one flag")
	}
	if s.VMs != nil && len(s.VMs) == 0 {
		return fmt.Errorf("at least one VM is required")
	}
	if err := s.Validate(); err != nil {
		return err
	}

	started := time.Now()
	spin := rumptyterm.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	req := api.UpdateLoadBalancerRequest{
		TargetPort:     s.Port,
		TargetProtocol: s.Protocol,
		HealthCheck:    s.healthCheck(),
		StickySessions: s.Sticky,
		AllowedCIDRs:   s.allowedCIDRs(),
	}
	if s.VMs != nil {
		vms, err := resolveVMs(ctx, rt, s.VMs)
		if err != nil {
			return err
		}
		uids := vmUIDs(vms)
		req.VMUIDs = &uids
	}

	spin.Update("Updating " + lbName(&target))
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().UpdateLoadBalancer(ctx, workspace, target.UID, &req, api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	lb, err := waitForChange(ctx, rt, &result, &target, "change", "to apply the change", spin)
	if err != nil {
		return err
	}
	spin.Stop()

	w := rt.Streams.Out
	fmt.Fprintln(w, rumptyterm.Success(w, fmt.Sprintf("Updated load balancer %s %s", lbName(&lb), rumptyterm.Muted(w, "("+formatElapsed(time.Since(started))+")"))))
	return nil
}

func Delete(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	spin := rumptyterm.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}
	spin.Stop()

	if !yes {
		ok, err := confirm(rt, fmt.Sprintf("Delete load balancer %s (%s)?", lbName(&target), target.URL))
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotConfirmed
		}
	}

	started := time.Now()
	spin = rumptyterm.StartSpinner(out(rt), "Deleting "+lbName(&target))
	defer spin.Stop()
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().DeleteLoadBalancer(ctx, workspace, target.UID, api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	if !result.IsCompleted && strings.TrimSpace(result.OperationID) != "" {
		spin.Update(fmt.Sprintf("Waiting for %s to be deleted", lbName(&target)))
		if err := waitForOperation(ctx, rt, result.OperationID); err != nil {
			return changeError(ctx, rt, &target, "delete", err)
		}
	}
	spin.Stop()

	w := rt.Streams.Out
	fmt.Fprintln(w, rumptyterm.Success(w, fmt.Sprintf("Deleted load balancer %s %s", lbName(&target), rumptyterm.Muted(w, "("+formatElapsed(time.Since(started))+")"))))
	return nil
}

func AddTargets(ctx context.Context, rt *app.Runtime, ref string, vmRefs []string) error {
	if len(vmRefs) == 0 {
		return fmt.Errorf("at least one VM is required")
	}

	spin := rumptyterm.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}
	vms, err := resolveVMs(ctx, rt, vmRefs)
	if err != nil {
		return err
	}

	names := strings.Join(vmSlugs(vms), ", ")
	spin.Update(fmt.Sprintf("Adding %s to %s", names, lbName(&target)))
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().AddLoadBalancerTargets(ctx, workspace, target.UID, vmUIDs(vms), api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	lb, err := waitForChange(ctx, rt, &result, &target, "change", "to apply the change", spin)
	if err != nil {
		return err
	}
	spin.Stop()

	w := rt.Streams.Out
	fmt.Fprintln(w, rumptyterm.Success(w, fmt.Sprintf("Added %s to %s", names, lbName(&lb))))
	fmt.Fprintf(w, "Targets: %s\n", targetNames(&lb))
	return nil
}

func RemoveTarget(ctx context.Context, rt *app.Runtime, ref, vmRef string) error {
	vmRef = strings.TrimSpace(vmRef)
	if vmRef == "" {
		return fmt.Errorf("vm name or uid is required")
	}

	spin := rumptyterm.StartSpinner(out(rt), "Finding "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}
	vmUID, vmName, err := findTarget(ctx, rt, &target, vmRef)
	if err != nil {
		return err
	}

	spin.Update(fmt.Sprintf("Removing %s from %s", vmName, lbName(&target)))
	workspace := strings.TrimSpace(rt.Config.Workspace)
	result, err := rt.API().RemoveLoadBalancerTarget(ctx, workspace, target.UID, vmUID, api.NewIdempotencyKey())
	if err != nil {
		return err
	}
	lb, err := waitForChange(ctx, rt, &result, &target, "change", "to apply the change", spin)
	if err != nil {
		return err
	}
	spin.Stop()

	w := rt.Streams.Out
	fmt.Fprintln(w, rumptyterm.Success(w, fmt.Sprintf("Removed %s from %s", vmName, lbName(&lb))))
	fmt.Fprintf(w, "Targets: %s\n", targetNames(&lb))
	return nil
}

// findTarget matches ref against the load balancer's targets, falling back to
// the workspace VM list so a VM slug works when the target shows its name.
func findTarget(ctx context.Context, rt *app.Runtime, lb *api.LoadBalancer, ref string) (uid, name string, err error) {
	for i := range lb.Targets {
		if lb.Targets[i].VMUID == ref || lb.Targets[i].VMName == ref {
			return lb.Targets[i].VMUID, ref, nil
		}
	}
	vms, err := resolveVMs(ctx, rt, []string{ref})
	if err != nil {
		return "", "", err
	}
	for i := range lb.Targets {
		if lb.Targets[i].VMUID == vms[0].UID {
			return vms[0].UID, vms[0].Slug, nil
		}
	}
	return "", "", fmt.Errorf("vm %s is not a target of %s", ref, lbName(lb))
}

func confirm(rt *app.Runtime, question string) (bool, error) {
	in := rt.Streams.In
	if in == nil {
		in = os.Stdin
	}
	if f, ok := in.(*os.File); ok && !term.IsTerminal(int(f.Fd())) { //nolint:gosec // stdin fd is small on supported platforms
		return false, config.NewUsageError("refusing to delete without confirmation; pass --yes")
	}
	return rumptyterm.Confirm(in, out(rt), question)
}
