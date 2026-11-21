package vm_test

import (
	"bytes"
	"context"
	"encoding/json"
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
			Slug:          "warm-jollof",
			DisplayStatus: "running",
			PlanSlug:      "medium",
			ImageSlug:     "ubuntu-24-04",
			AppURL:        "https://warm-jollof.app.stg.rumptycloud.com",
		},
		{
			Slug:   "dev-box",
			Status: "stopped",
			Kind:   "persistent",
		},
	}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.List(context.Background(), rt, term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "NAME")
	assert.Contains(t, s, "PLAN")
	assert.Contains(t, s, "IMAGE")
	assert.Contains(t, s, "APP URL")
	assert.NotContains(t, s, "UPTIME")
	assert.Contains(t, s, "warm-jollof")
	assert.Contains(t, s, "medium")
	assert.Contains(t, s, "ubuntu-24-04")
	assert.Contains(t, s, "https://warm-jollof.app.stg.rumptycloud.com")
	assert.Contains(t, s, "running")
	assert.Contains(t, s, "stopped")
	assert.Contains(t, s, "dev-box")
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

	require.NoError(t, vm.List(context.Background(), rt, term.OutputTable))
	assert.Contains(t, strings.TrimSpace(out.String()), "No VMs in acme.")
}

func TestList_json(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{{UID: "vm-1", Slug: "web", DisplayStatus: "running"}}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, vm.List(context.Background(), rt, term.OutputJSON))
	var got []api.VM
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "web", got[0].Slug)
}
