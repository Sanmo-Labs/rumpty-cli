package commands

import (
	"context"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func isolateCreds(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(config.EnvToken, "")
	t.Setenv(config.EnvWorkspace, "")
	t.Setenv(config.EnvAPIURL, "")
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

func newRuntime(t *testing.T, cfg *config.Config) (*app.Runtime, *mocks.MockClientAPI) {
	t.Helper()
	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	rt := &app.Runtime{
		Config:    cfg,
		Streams:   term.Streams{Out: io.Discard, ErrOut: io.Discard},
		APIClient: mock,
	}
	return rt, mock
}

func TestCompleteVMNames_returnsSlugs(t *testing.T) {
	isolateCreds(t)
	rt, mock := newRuntime(t, &config.Config{Token: "tok", Workspace: "acme"})
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{
		{Slug: "warm-jollof"},
		{Slug: "dev-box"},
		{Slug: ""}, // blank slugs are skipped
	}, nil)

	got, directive := completeVMNames(rt)(newCompletionCmd(), nil, "")

	assert.Equal(t, []string{"warm-jollof", "dev-box"}, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompleteVMNames_fallsBackToDefaultWorkspace(t *testing.T) {
	isolateCreds(t)
	// No workspace configured: completer should resolve the default workspace.
	rt, mock := newRuntime(t, &config.Config{Token: "tok"})
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{Slug: "acme-dev"},
		{Slug: "acme-prod", IsDefault: true},
	}, nil)
	mock.EXPECT().ListVMs(gomock.Any(), "acme-prod").Return([]api.VM{{Slug: "dev-box"}}, nil)

	got, directive := completeVMNames(rt)(newCompletionCmd(), nil, "")

	assert.Equal(t, []string{"dev-box"}, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompleteVMNames_ambiguousWorkspaceReturnsError(t *testing.T) {
	isolateCreds(t)
	// No workspace and no default among several: can't disambiguate.
	rt, mock := newRuntime(t, &config.Config{Token: "tok"})
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{Slug: "acme-dev"},
		{Slug: "acme-prod"},
	}, nil)

	got, directive := completeVMNames(rt)(newCompletionCmd(), nil, "")

	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}

func TestCompleteVMNames_onlyCompletesFirstArg(t *testing.T) {
	isolateCreds(t)
	rt, _ := newRuntime(t, &config.Config{Token: "tok", Workspace: "acme"})

	got, directive := completeVMNames(rt)(newCompletionCmd(), []string{"warm-jollof"}, "")

	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompleteVMNames_missingAuthReturnsError(t *testing.T) {
	isolateCreds(t)
	rt, _ := newRuntime(t, &config.Config{})

	got, directive := completeVMNames(rt)(newCompletionCmd(), nil, "")

	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}

func TestCompleteVMNames_apiErrorReturnsError(t *testing.T) {
	isolateCreds(t)
	rt, mock := newRuntime(t, &config.Config{Token: "tok", Workspace: "acme"})
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return(nil, assert.AnError)

	got, directive := completeVMNames(rt)(newCompletionCmd(), nil, "")

	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}

func TestCompleteWorkspaceSlugs_returnsSlugs(t *testing.T) {
	isolateCreds(t)
	rt, mock := newRuntime(t, &config.Config{Token: "tok"})
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{Slug: "acme-dev"},
		{Slug: "acme-prod"},
	}, nil)

	got, directive := completeWorkspaceSlugs(rt)(newCompletionCmd(), nil, "")

	assert.Equal(t, []string{"acme-dev", "acme-prod"}, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompleteWorkspaceSlugs_missingTokenReturnsError(t *testing.T) {
	isolateCreds(t)
	rt, _ := newRuntime(t, &config.Config{})

	got, directive := completeWorkspaceSlugs(rt)(newCompletionCmd(), nil, "")

	assert.Nil(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}
