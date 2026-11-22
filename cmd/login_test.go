package commands

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func TestLogin_tokenStdin(t *testing.T) {
	isolateCreds(t)
	rt, mock := newRuntime(t, &config.Config{APIURL: "https://api.example"})
	rt.Streams = term.Streams{In: strings.NewReader("  rumpty_secret\n"), Out: io.Discard, ErrOut: io.Discard}
	mock.EXPECT().Me(gomock.Any()).Return(api.User{Username: "alice"}, nil)

	cmd := newLoginCmd(rt)
	cmd.SetArgs([]string{"--token-stdin"})
	require.NoError(t, cmd.Execute())

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "rumpty_secret", creds.Token)
}

func TestLogin_tokenStdinEmpty(t *testing.T) {
	isolateCreds(t)
	rt, _ := newRuntime(t, &config.Config{APIURL: "https://api.example"})
	rt.Streams = term.Streams{In: strings.NewReader("\n"), Out: io.Discard, ErrOut: io.Discard}

	cmd := newLoginCmd(rt)
	cmd.SetArgs([]string{"--token-stdin"})
	cmd.SilenceUsage = true
	require.ErrorContains(t, cmd.Execute(), "no API key on stdin")
}

func TestLogin_tokenAndTokenStdinConflict(t *testing.T) {
	isolateCreds(t)
	rt, _ := newRuntime(t, &config.Config{APIURL: "https://api.example"})

	cmd := newLoginCmd(rt)
	cmd.SetArgs([]string{"--token", "x", "--token-stdin"})
	cmd.SilenceUsage = true
	require.ErrorContains(t, cmd.Execute(), "not both")
}
