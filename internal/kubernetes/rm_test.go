package kubernetes_test

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
	"github.com/cloverworks-hq/rumpty-cli/internal/kubernetes"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newK8sRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testCluster() api.Cluster {
	return api.Cluster{UID: "k8s-1", Name: "staging", Slug: "staging", Status: "running", ZoneSlug: "olas-closet", Version: "1.34.2"}
}

func TestRm_withYesDeletesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListClusters(gomock.Any(), "prod-team").Return([]api.Cluster{testCluster()}, nil)
	mock.EXPECT().DeleteCluster(gomock.Any(), "prod-team", "k8s-1").Return(api.ClusterOperationResult{
		OperationID: "op-1",
		Cluster:     testCluster(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{UID: "op-1", Status: "succeeded"}, nil)

	rt, out := newK8sRuntime(t, mock)

	require.NoError(t, kubernetes.Rm(context.Background(), rt, "staging", true))
	assert.Contains(t, out.String(), "Deleted cluster staging")
}

func TestRm_operationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListClusters(gomock.Any(), "prod-team").Return([]api.Cluster{testCluster()}, nil)
	mock.EXPECT().DeleteCluster(gomock.Any(), "prod-team", "k8s-1").Return(api.ClusterOperationResult{
		OperationID: "op-1",
		Cluster:     testCluster(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{UID: "op-1", Status: "failed"}, nil)

	rt, _ := newK8sRuntime(t, mock)

	err := kubernetes.Rm(context.Background(), rt, "staging", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed")
}

func TestRm_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListClusters(gomock.Any(), "prod-team").Return([]api.Cluster{}, nil)

	rt, _ := newK8sRuntime(t, mock)

	err := kubernetes.Rm(context.Background(), rt, "missing", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no cluster")
}

func TestRm_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListClusters(gomock.Any(), "prod-team").Return([]api.Cluster{testCluster()}, nil)

	rt, _ := newK8sRuntime(t, mock)

	err := kubernetes.Rm(context.Background(), rt, "staging", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}
