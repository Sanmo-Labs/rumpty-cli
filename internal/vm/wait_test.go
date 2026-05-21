package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestVMHasTargetStatus(t *testing.T) {
	t.Parallel()

	stopped := api.VM{DisplayStatus: "stopped"}
	running := api.VM{DisplayStatus: "running"}
	assert.True(t, vmHasTargetStatus("stop", &stopped))
	assert.True(t, vmHasTargetStatus("start", &api.VM{Status: "running"}))
	assert.False(t, vmHasTargetStatus("stop", &running))
}
