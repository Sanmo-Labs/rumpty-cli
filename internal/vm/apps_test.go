package vm_test

import (
	"bytes"
	"context"
	"io"
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

func TestListApps_printsApps(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	target := api.VM{UID: "vm-uid-7", Slug: "test-vm8", Name: "test-vm8"}
	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{target}, nil),
		mock.EXPECT().ListVMApps(gomock.Any(), "acme", "vm-uid-7").Return([]api.VMApp{
			{Slug: "openclaw", Port: 18790, Status: "ready", URL: "https://openclaw.app.stg.rumptycloud.com"},
		}, nil),
	)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.ListApps(context.Background(), rt, "test-vm8"))
	assert.Contains(t, out.String(), "NAME")
	assert.Contains(t, out.String(), "PORT")
	assert.Contains(t, out.String(), "openclaw")
	assert.Contains(t, out.String(), "18790")
	assert.Contains(t, out.String(), "https://openclaw.app.stg.rumptycloud.com")
}

func TestListApps_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	target := api.VM{UID: "vm-uid-7", Slug: "test-vm8", Name: "test-vm8"}
	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{target}, nil),
		mock.EXPECT().ListVMApps(gomock.Any(), "acme", "vm-uid-7").Return([]api.VMApp{}, nil),
	)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.ListApps(context.Background(), rt, "test-vm8"))
	assert.Contains(t, out.String(), "No exposed apps for test-vm8.")
}
