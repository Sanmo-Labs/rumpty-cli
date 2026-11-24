package kubernetes

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func TestKubeconfig_replacesExistingFileWithPrivatePermissions(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListClusters(gomock.Any(), "").Return([]api.Cluster{
		{UID: "c-1", Name: "prod", Slug: "prod"},
	}, nil)
	mock.EXPECT().GetCluster(gomock.Any(), "", "c-1").Return(api.Cluster{
		UID:        "c-1",
		Name:       "prod",
		Slug:       "prod",
		Kubeconfig: "apiVersion: v1",
	}, nil)

	t.Chdir(t.TempDir())
	path := "prod-kubeconfig.yaml"
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: &bytes.Buffer{}, ErrOut: io.Discard},
		APIClient: mock,
	}
	require.NoError(t, Kubeconfig(context.Background(), rt, "prod", path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "apiVersion: v1", string(data))
}
