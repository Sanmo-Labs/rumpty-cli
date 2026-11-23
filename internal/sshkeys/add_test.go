package sshkeys_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/sshkeys"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

const testPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIK0wmN/Cr3JXqmLW7u+g9pTh+wyqDHpSQEIQczXkVx9q ada@laptop"

func newAddRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: out},
		APIClient: mock,
	}, out
}

func writeKeyFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "id_ed25519.pub")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestAdd_addsKeyUsingCommentAsName(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	path := writeKeyFile(t, testPublicKey+"\n")
	mock.EXPECT().AddSSHKey(gomock.Any(), "prod-team", api.AddSSHKeyRequest{
		Name:      "ada@laptop",
		PublicKey: testPublicKey,
	}).Return(api.SSHKey{UID: "key-1", Name: "ada@laptop", Fingerprint: "SHA256:abc"}, nil)

	rt, out := newAddRuntime(t, mock)

	require.NoError(t, sshkeys.Add(context.Background(), rt, path, ""))
	assert.Contains(t, out.String(), "Added ada@laptop (SHA256:abc)")
}

func TestAdd_nameFlagOverridesComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	path := writeKeyFile(t, testPublicKey+"\n")
	mock.EXPECT().AddSSHKey(gomock.Any(), "prod-team", api.AddSSHKeyRequest{
		Name:      "laptop",
		PublicKey: testPublicKey,
	}).Return(api.SSHKey{UID: "key-1", Name: "laptop", Fingerprint: "SHA256:abc"}, nil)

	rt, _ := newAddRuntime(t, mock)

	require.NoError(t, sshkeys.Add(context.Background(), rt, path, "laptop"))
}

func TestAdd_rejectsPrivateKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	path := writeKeyFile(t, "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXk\n-----END OPENSSH PRIVATE KEY-----\n")

	rt, _ := newAddRuntime(t, mock)

	err := sshkeys.Add(context.Background(), rt, path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key")
}

func TestAdd_rejectsInvalidKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	path := writeKeyFile(t, "not an ssh key\n")

	rt, _ := newAddRuntime(t, mock)

	err := sshkeys.Add(context.Background(), rt, path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ssh public key")
}

func TestAdd_missingFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newAddRuntime(t, mock)

	err := sshkeys.Add(context.Background(), rt, filepath.Join(t.TempDir(), "nope.pub"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope.pub")
}
