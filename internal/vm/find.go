package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
)

func Find(ctx context.Context, rt *app.Runtime, ref string) (api.VM, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return api.VM{}, fmt.Errorf("vm name or slug is required")
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	vms, err := rt.API().ListVMs(ctx, workspace)
	if err != nil {
		return api.VM{}, err
	}

	for i := range vms {
		if vms[i].Slug == ref || vms[i].UID == ref || vms[i].Name == ref {
			return vms[i], nil
		}
	}
	return api.VM{}, fmt.Errorf("vm %q not found in workspace %s", ref, workspace)
}
