package volume_test

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
	"github.com/cloverworks-hq/rumpty-cli/internal/volume"
)

func newVolumeRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testVolume() api.Volume {
	return api.Volume{UID: "vol-1", Name: "data", Slug: "data", Status: "available", ZoneSlug: "olas-closet", SizeGiB: 20, Filesystem: "ext4"}
}

func TestCreate_rejectsInvalidSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newVolumeRuntime(t, mock)

	err := volume.Create(context.Background(), rt, "data", 0, "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 1024")
}

func TestCreate_waitsAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().CreateVolume(gomock.Any(), "prod-team", gomock.Any()).Return(api.VolumeOperationResult{
		OperationID: "op-1",
		Volume:      testVolume(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{UID: "op-1", Status: "succeeded"}, nil)

	rt, out := newVolumeRuntime(t, mock)

	require.NoError(t, volume.Create(context.Background(), rt, "data", 20, "", "", ""))
	assert.Contains(t, out.String(), "Created volume data (20 GiB, olas-closet)")
}

func TestCreate_operationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().CreateVolume(gomock.Any(), "prod-team", gomock.Any()).Return(api.VolumeOperationResult{
		OperationID: "op-1",
		Volume:      testVolume(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{UID: "op-1", Status: "failed"}, nil)

	rt, _ := newVolumeRuntime(t, mock)

	err := volume.Create(context.Background(), rt, "data", 20, "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed")
}

func TestRm_withYesDeletesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVolumes(gomock.Any(), "prod-team").Return([]api.Volume{testVolume()}, nil)
	mock.EXPECT().DeleteVolume(gomock.Any(), "prod-team", "vol-1").Return(api.VolumeOperationResult{
		OperationID: "op-2",
		Volume:      testVolume(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-2").Return(api.Operation{UID: "op-2", Status: "succeeded"}, nil)

	rt, out := newVolumeRuntime(t, mock)

	require.NoError(t, volume.Rm(context.Background(), rt, "data", true))
	assert.Contains(t, out.String(), "Deleted volume data")
}

func TestRm_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVolumes(gomock.Any(), "prod-team").Return([]api.Volume{testVolume()}, nil)

	rt, _ := newVolumeRuntime(t, mock)

	err := volume.Rm(context.Background(), rt, "data", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestRm_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVolumes(gomock.Any(), "prod-team").Return([]api.Volume{}, nil)

	rt, _ := newVolumeRuntime(t, mock)

	err := volume.Rm(context.Background(), rt, "missing", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no volume")
}
