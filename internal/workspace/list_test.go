package workspace_test

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
	"github.com/Sanmo-Labs/rumpty-cli/internal/workspace"
)

func TestList_printsWorkspaces(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{Name: "Production", Slug: "production-team", IsDefault: true},
		{Name: "Dev", Slug: "acme-dev"},
	}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, workspace.List(context.Background(), rt))
	s := out.String()
	assert.Contains(t, s, "SLUG")
	assert.Contains(t, s, "NAME")
	assert.Contains(t, s, "production-team")
	assert.Contains(t, s, "acme-dev")
	assert.NotContains(t, s, "DEFAULT")
}

func TestList_empty(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, workspace.List(context.Background(), rt))
	assert.Contains(t, strings.TrimSpace(out.String()), "No workspaces.")
}
