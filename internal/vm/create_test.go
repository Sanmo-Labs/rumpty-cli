package vm_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
	"github.com/cloverworks-hq/rumpty-cli/internal/vm"
)

func TestCreate_waitChecksNewVMByUID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	req := &api.CreateVMRequest{
		Name: "web", PlanSlug: "micro", ImageSlug: "ubuntu-2604-lts", VPCSlug: "default", SSHKeyUIDs: []string{"key-1"},
	}
	gomock.InOrder(
		mock.EXPECT().CreateVM(gomock.Any(), "acme", req, gomock.Any()).Return(api.VMOperationResult{
			VMUID: "vm-1", OperationID: "op-1", Status: "queued",
		}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-1").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").DoAndReturn(func(context.Context, string) ([]api.VM, error) {
			cancel()
			return []api.VM{
				{UID: "other-vm", Slug: "vm-1", Status: "running"},
				{UID: "vm-1", Slug: "web", Status: "provisioning"},
			}, nil
		}),
	)
	var out bytes.Buffer
	rt := &app.Runtime{
		Config:    &config.Config{Workspace: "acme"},
		Streams:   term.Streams{Out: &out, ErrOut: io.Discard},
		APIClient: mock,
	}
	require.ErrorIs(t, vm.Create(ctx, rt, req, true), context.Canceled)
	assert.Contains(t, out.String(), "operation op-1")
	assert.NotContains(t, out.String(), "is running")
}
