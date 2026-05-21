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

func TestStop_waitsUntilStopped(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	running := api.VM{UID: "vm-uid-7", Slug: "test-vm7", Name: "test-vm7", DisplayStatus: "running"}
	stopped := api.VM{UID: "vm-uid-7", Slug: "test-vm7", Name: "test-vm7", DisplayStatus: "stopped"}

	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{running}, nil),
		mock.EXPECT().StopVM(gomock.Any(), "acme", "vm-uid-7", gomock.Any()).Return(api.VMOperationResult{
			OperationID: "op-1",
			Action:      "stop",
		}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{running}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-1").Return(api.Operation{Status: "running"}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-1").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{stopped}, nil),
	)

	errOut := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: io.Discard, ErrOut: errOut},
		APIClient: mock,
	}

	require.NoError(t, vm.Stop(context.Background(), rt, "test-vm7"))
	assert.Contains(t, errOut.String(), "Stopped test-vm7")
}

func TestStop_alreadyStopped(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{
		{UID: "vm-uid-7", Slug: "test-vm7", DisplayStatus: "stopped"},
	}, nil)

	errOut := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Workspace: "acme"},
		Streams:   term.Streams{ErrOut: errOut},
		APIClient: mock,
	}

	require.NoError(t, vm.Stop(context.Background(), rt, "test-vm7"))
	assert.Contains(t, errOut.String(), "Stopped test-vm7")
}
