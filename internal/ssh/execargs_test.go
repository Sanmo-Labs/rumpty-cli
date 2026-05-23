package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestParseExecArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		vm      string
		command []string
		err     string
	}{
		{
			name:    "explicit separator",
			args:    []string{"my-vm", "--", "uptime"},
			vm:      "my-vm",
			command: []string{"uptime"},
		},
		{
			name:    "cobra strips separator",
			args:    []string{"my-vm", "uptime"},
			vm:      "my-vm",
			command: []string{"uptime"},
		},
		{
			name:    "multi word",
			args:    []string{"my-vm", "bash", "-lc", "echo hi"},
			vm:      "my-vm",
			command: []string{"bash", "-lc", "echo hi"},
		},
		{
			name: "missing command",
			args: []string{"my-vm"},
			err:  "remote command is required",
		},
		{
			name: "empty command after separator",
			args: []string{"my-vm", "--"},
			err:  "remote command is required",
		},
		{
			name: "missing vm",
			args: []string{},
			err:  "vm name or slug is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vm, cmd, err := rumptyssh.ParseExecArgs(tt.args)
			if tt.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.vm, vm)
			assert.Equal(t, tt.command, cmd)
		})
	}
}
