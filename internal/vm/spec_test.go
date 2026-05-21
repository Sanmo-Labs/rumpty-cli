package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestFormatSpec_withPlan(t *testing.T) {
	t.Parallel()
	v := api.VM{
		PlanSlug:  "micro",
		VCPU:      1,
		MemoryMiB: 1024,
		DiskGiB:   20,
		ImageName: "Ubuntu 24.04 LTS",
	}
	got := formatSpec(&v)
	assert.Equal(t, "micro · 1 vCPU · 1GiB · 20GiB disk · Ubuntu 24.04 LTS", got)
}

func TestFormatSpec_withPlan_imageSlugFallback(t *testing.T) {
	t.Parallel()
	v := api.VM{
		PlanSlug:  "micro",
		VCPU:      1,
		MemoryMiB: 1024,
		DiskGiB:   10,
		ImageSlug: "ubuntu-24-04",
	}
	got := formatSpec(&v)
	assert.Equal(t, "micro · 1 vCPU · 1GiB · 10GiB disk · ubuntu-24-04", got)
}

func TestFormatSpec_fallback(t *testing.T) {
	t.Parallel()
	v := api.VM{
		Kind:      "persistent",
		DiskGiB:   20,
		ZoneSlug:  "olas-closet",
		ImageName: "Ubuntu 24.04 LTS",
	}
	got := formatSpec(&v)
	assert.Equal(t, "persistent · 20GiB disk · olas-closet · Ubuntu 24.04 LTS", got)
}
