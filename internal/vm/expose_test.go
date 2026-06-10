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

func TestExpose_waitsAndPrintsURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	target := api.VM{UID: "vm-uid-7", Slug: "test-vm7", Name: "test-vm7", DisplayStatus: "running"}

	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{target}, nil),
		mock.EXPECT().ExposeVMApp(gomock.Any(), "acme", "vm-uid-7", api.ExposeVMAppRequest{
			Name: "openclaw",
			Port: 18789,
		}, gomock.Any()).Return(api.ExposeVMAppResult{
			OperationID: "op-3",
			VMUID:       "vm-uid-7",
			VMName:      "test-vm7",
			Status:      "queued",
			App: api.VMApp{
				Slug: "openclaw",
				Port: 18789,
				URL:  "https://openclaw-abc.app.stg.rumptycloud.com",
			},
		}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-3").Return(api.Operation{Status: "succeeded"}, nil),
	)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.Expose(context.Background(), rt, "test-vm7", 18789, "openclaw"))
	assert.Contains(t, out.String(), "Exposed openclaw on test-vm7")
	assert.Contains(t, out.String(), "URL: https://openclaw-abc.app.stg.rumptycloud.com")
	assert.Contains(t, out.String(), "View: rumpty vm expose ls test-vm7 --ws acme")
	assert.Contains(t, out.String(), "Remove: rumpty unexpose test-vm7 --ws acme --name openclaw")
}

func TestExpose_rejectsInvalidPort(t *testing.T) {
	rt := &app.Runtime{Config: &config.Config{Workspace: "acme"}}
	require.ErrorContains(t, vm.Expose(context.Background(), rt, "test-vm7", 0, ""), "port must be between")
}
