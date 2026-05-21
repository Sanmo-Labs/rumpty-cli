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
	}
	got := formatSpec(&v)
	assert.Equal(t, "micro · 1 vCPU · 1GiB · 20GiB disk", got)
}

func TestFormatSpec_fallback(t *testing.T) {
	t.Parallel()
	v := api.VM{
		Kind:     "persistent",
		DiskGiB:  20,
		ZoneSlug: "olas-closet",
	}
	got := formatSpec(&v)
	assert.Equal(t, "persistent · 20GiB disk · olas-closet", got)
}
