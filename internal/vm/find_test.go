package vm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func TestFind_bySlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{
		{UID: "vm-uid-7", Slug: "test-vm7", Name: "test-vm7"},
	}, nil)

	rt := &app.Runtime{
		Config:    &config.Config{Workspace: "acme"},
		APIClient: mock,
	}
	got, err := vm.Find(context.Background(), rt, "test-vm7")
	require.NoError(t, err)
	assert.Equal(t, "vm-uid-7", got.UID)
}

func TestFind_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{}, nil)

	rt := &app.Runtime{
		Config:    &config.Config{Workspace: "acme"},
		APIClient: mock,
	}
	_, err := vm.Find(context.Background(), rt, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
