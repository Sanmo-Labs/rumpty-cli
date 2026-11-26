package lb

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
)

// Find returns the load balancer whose slug, name or uid matches ref.
func Find(ctx context.Context, rt *app.Runtime, ref string) (api.LoadBalancer, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return api.LoadBalancer{}, fmt.Errorf("load balancer name or uid is required")
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	lbs, err := rt.API().ListLoadBalancers(ctx, workspace)
	if err != nil {
		return api.LoadBalancer{}, err
	}
	for i := range lbs {
		if lbs[i].Slug == ref || lbs[i].UID == ref || lbs[i].Name == ref {
			return lbs[i], nil
		}
	}
	return api.LoadBalancer{}, fmt.Errorf("load balancer %q not found in workspace %s", ref, workspace)
}

// resolveVMs maps VM names, slugs or uids to VMs with a single list call.
func resolveVMs(ctx context.Context, rt *app.Runtime, refs []string) ([]api.VM, error) {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	vms, err := rt.API().ListVMs(ctx, workspace)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]api.VM, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("vm name or uid is required")
		}
		found := false
		for i := range vms {
			if vms[i].Slug == ref || vms[i].UID == ref || vms[i].Name == ref {
				if !seen[vms[i].UID] {
					seen[vms[i].UID] = true
					out = append(out, vms[i])
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("vm %q not found in workspace %s", ref, workspace)
		}
	}
	if len(out) > MaxTargets {
		return nil, fmt.Errorf("a load balancer can have at most %d VMs", MaxTargets)
	}
	return out, nil
}

func vmUIDs(vms []api.VM) []string {
	uids := make([]string, len(vms))
	for i := range vms {
		uids[i] = vms[i].UID
	}
	return uids
}

func vmSlugs(vms []api.VM) []string {
	slugs := make([]string, len(vms))
	for i := range vms {
		slugs[i] = vms[i].Slug
	}
	return slugs
}
