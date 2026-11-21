package sync

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
)

func TestRevokeSavedKeys(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, saveBucketKey("b1", &BucketKey{UID: "key-1", Workspace: "acme", AccessKeyID: "AK1", Bucket: "backups"}))
	require.NoError(t, saveBucketKey("b2", &BucketKey{UID: "key-2", Workspace: "acme", AccessKeyID: "AK2", Bucket: "photos"}))
	require.NoError(t, saveBucketKey("b3", &BucketKey{AccessKeyID: "AK3", Bucket: "legacy"}))

	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	mock.EXPECT().RevokeAssetAccessKey(gomock.Any(), "acme", "key-1").Return(nil)
	mock.EXPECT().RevokeAssetAccessKey(gomock.Any(), "acme", "key-2").Return(errors.New("forbidden"))

	notRevoked, err := RevokeSavedKeys(context.Background(), mock)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"access key AK2 for bucket photos",
		"access key AK3 for bucket legacy",
	}, notRevoked)

	path, err := keysFilePath()
	require.NoError(t, err)
	assert.NoFileExists(t, path)
}

func TestRevokeSavedKeys_withoutSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, saveBucketKey("b1", &BucketKey{UID: "key-1", Workspace: "acme", AccessKeyID: "AK1", Bucket: "backups"}))

	notRevoked, err := RevokeSavedKeys(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"access key AK1 for bucket backups"}, notRevoked)

	path, err := keysFilePath()
	require.NoError(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}
