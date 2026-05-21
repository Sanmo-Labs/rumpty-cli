package vm

import (
	"fmt"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func formatSpec(v *api.VM) string {
	osName := formatOS(v)

	if v.PlanSlug != "" && v.VCPU > 0 {
		spec := fmt.Sprintf("%s · %d vCPU · %s · %dGiB disk",
			v.PlanSlug, v.VCPU, formatMemory(v.MemoryMiB), v.DiskGiB)
		if osName != "" {
			spec += " · " + osName
		}
		return spec
	}

	var parts []string
	if k := strings.TrimSpace(v.Kind); k != "" {
		parts = append(parts, k)
	}
	if v.DiskGiB > 0 {
		parts = append(parts, fmt.Sprintf("%dGiB disk", v.DiskGiB))
	}
	if z := strings.TrimSpace(v.ZoneSlug); z != "" {
		parts = append(parts, z)
	}
	if osName != "" {
		parts = append(parts, osName)
	}
	return strings.Join(parts, " · ")
}

func formatOS(v *api.VM) string {
	if name := strings.TrimSpace(v.ImageName); name != "" {
		return name
	}
	return strings.TrimSpace(v.ImageSlug)
}

func formatMemory(mib int) string {
	if mib <= 0 {
		return ""
	}
	if mib >= 1024 && mib%1024 == 0 {
		return fmt.Sprintf("%dGiB", mib/1024)
	}
	return fmt.Sprintf("%dMiB", mib)
}
