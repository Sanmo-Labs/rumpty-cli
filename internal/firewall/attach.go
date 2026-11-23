package firewall

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// validResourceTypes lists the resource kinds a firewall policy attaches to.
var validResourceTypes = []string{"vm", "database", "deployment"}

// Attach attaches a firewall policy to a VM, database or deployment. The
// resource is resolved from the workspace's VMs and databases by uid, name
// or slug; for a deployment pass --type deployment and its uid.
func Attach(ctx context.Context, rt *app.Runtime, policyRef, resourceRef, resourceType string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding firewall policy")
	policies, err := rt.API().ListFirewallPolicies(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	policy, err := findPolicy(policies, policyRef, workspace)
	if err != nil {
		return err
	}

	kind, uid, name, err := resolveResource(ctx, rt, workspace, resourceRef, resourceType)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Attaching "+policy.Name+" to "+name)
	result, err := rt.API().AttachFirewallPolicy(ctx, workspace, policy.UID, api.FirewallAttachRequest{
		ResourceType: kind,
		ResourceUID:  uid,
	})
	if err != nil {
		spin.Stop()
		return err
	}
	if result.OperationID == "" {
		spin.Stop()
		fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Policy %s is already attached to %s %s", policy.Name, kind, name)))
		return nil
	}
	if err := waitForPolicyApply(ctx, rt, workspace, result.OperationID, spin); err != nil {
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Attached policy %s to %s %s", policy.Name, kind, name)))
	return nil
}

// Detach removes one attachment of a firewall policy, matched by the
// resource's uid, name or slug.
func Detach(ctx context.Context, rt *app.Runtime, policyRef, resourceRef, resourceType string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding firewall policy")
	policies, err := rt.API().ListFirewallPolicies(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	policy, err := findPolicy(policies, policyRef, workspace)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Reading the policy's attachments")
	full, err := rt.API().GetFirewallPolicy(ctx, workspace, policy.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	if resourceType != "" && !validType(resourceType) {
		return config.NewUsageError("--type must be one of %s", strings.Join(validResourceTypes, ", "))
	}
	attachment, err := findAttachment(full.AttachedResources, resourceRef, resourceType, workspace)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Detaching "+policy.Name+" from "+attachment.Resource.Name)
	if err := rt.API().DetachFirewallPolicy(ctx, workspace, policy.UID, attachment.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Detached policy %s from %s %s", policy.Name, attachment.ResourceType, attachment.Resource.Name)))
	return nil
}

func validType(t string) bool {
	for _, k := range validResourceTypes {
		if k == t {
			return true
		}
	}
	return false
}

// resolveResource turns a resource reference into its kind, uid and name.
// An empty resourceType means auto-detect: VMs first, then databases, and
// deployments only with --type deployment (their uid passed directly).
func resolveResource(ctx context.Context, rt *app.Runtime, workspace, resourceRef, resourceType string) (string, string, string, error) {
	switch resourceType {
	case "":
		spin := term.StartLoading(errOut(rt), "Finding the resource")
		vms, err := rt.API().ListVMs(ctx, workspace)
		if err == nil {
			if vm, mErr := findVM(vms, resourceRef, workspace); mErr == nil {
				spin.Stop()
				return "vm", vm.UID, vm.Name, nil
			}
		}
		dbs, err := rt.API().ListDatabases(ctx, workspace)
		spin.Stop()
		if err != nil {
			return "", "", "", err
		}
		if db, dErr := findDatabase(dbs, resourceRef, workspace); dErr == nil {
			return "database", db.UID, db.Name, nil
		}
		return "", "", "", fmt.Errorf("no VM or database %q in workspace %s; for a deployment pass --type deployment and its uid", resourceRef, workspace)
	case "vm":
		spin := term.StartLoading(errOut(rt), "Finding the VM")
		vms, err := rt.API().ListVMs(ctx, workspace)
		spin.Stop()
		if err != nil {
			return "", "", "", err
		}
		vm, err := findVM(vms, resourceRef, workspace)
		if err != nil {
			return "", "", "", err
		}
		return "vm", vm.UID, vm.Name, nil
	case "database":
		spin := term.StartLoading(errOut(rt), "Finding the database")
		dbs, err := rt.API().ListDatabases(ctx, workspace)
		spin.Stop()
		if err != nil {
			return "", "", "", err
		}
		db, err := findDatabase(dbs, resourceRef, workspace)
		if err != nil {
			return "", "", "", err
		}
		return "database", db.UID, db.Name, nil
	case "deployment":
		return "deployment", resourceRef, resourceRef, nil
	default:
		return "", "", "", config.NewUsageError("--type must be one of %s", strings.Join(validResourceTypes, ", "))
	}
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

func findDatabase(dbs []api.Database, ref, workspaceSlug string) (api.Database, error) {
	var matches []api.Database
	for i := range dbs {
		d := &dbs[i]
		if d.UID == ref || d.Name == ref || d.Slug == ref {
			matches = append(matches, *d)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Database{}, fmt.Errorf("no database %q in workspace %s", ref, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.Database{}, fmt.Errorf("multiple databases match %q: %s", ref, strings.Join(names, ", "))
	}
}

func findAttachment(attachments []api.FirewallAttachment, resourceRef, resourceType, workspaceSlug string) (api.FirewallAttachment, error) {
	var matches []api.FirewallAttachment
	for i := range attachments {
		a := &attachments[i]
		if resourceType != "" && !strings.EqualFold(a.ResourceType, resourceType) {
			continue
		}
		r := &a.Resource
		if r.UID == resourceRef || r.Name == resourceRef || r.Slug == resourceRef {
			matches = append(matches, *a)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		if resourceType != "" {
			return api.FirewallAttachment{}, fmt.Errorf("policy is not attached to a %s %q in workspace %s", resourceType, resourceRef, workspaceSlug)
		}
		return api.FirewallAttachment{}, fmt.Errorf("policy has no attachment for %q in workspace %s", resourceRef, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Resource.Name)
		}
		return api.FirewallAttachment{}, fmt.Errorf("multiple attachments match %q: %s", resourceRef, strings.Join(names, ", "))
	}
}
