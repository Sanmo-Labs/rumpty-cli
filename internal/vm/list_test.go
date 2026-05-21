package vm_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func TestList_printsVMs(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{
		{
			Name:          "Test VM 7",
			Slug:          "test-vm7",
			DisplayStatus: "running",
			PlanSlug:      "micro",
			VCPU:          1,
			MemoryMiB:     1024,
			DiskGiB:       20,
		},
		{Name: "Dev box", Slug: "dev-box", Status: "stopped", Kind: "persistent", DiskGiB: 10, ZoneSlug: "olas-closet"},
	}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.List(context.Background(), rt))
	s := out.String()
	assert.Contains(t, s, "SLUG")
	assert.Contains(t, s, "SPEC")
	assert.Contains(t, s, "test-vm7")
	assert.Contains(t, s, "micro · 1 vCPU")
	assert.Contains(t, s, "stopped")
}

func TestList_empty(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.List(context.Background(), rt))
	assert.Contains(t, strings.TrimSpace(out.String()), "No VMs.")
}
