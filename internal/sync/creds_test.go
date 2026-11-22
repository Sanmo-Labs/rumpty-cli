package sync

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
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

func TestBucketKey_secretInKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer credentials.UseMockKeychain(nil)()

	key := BucketKey{UID: "k1", Workspace: "acme", AccessKeyID: "AK1", SecretAccessKey: "bucket-secret", Endpoint: "https://s3.example", Bucket: "backups"}
	require.NoError(t, saveBucketKey("b1", &key))

	path, err := keysFilePath()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "bucket-secret")

	loaded, ok := loadBucketKey("b1")
	require.True(t, ok)
	assert.Equal(t, "bucket-secret", loaded.SecretAccessKey)

	dropBucketKey("b1")
	_, ok = credentials.LoadSecret(bucketKeyAccount("b1"))
	assert.False(t, ok)
}

func TestBucketKey_movesFileSecretIntoKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(credentials.EnvNoKeychain, "1")
	key := BucketKey{AccessKeyID: "AK1", SecretAccessKey: "old-secret", Endpoint: "https://s3.example", Bucket: "backups"}
	require.NoError(t, saveBucketKey("b1", &key))

	t.Setenv(credentials.EnvNoKeychain, "")
	defer credentials.UseMockKeychain(nil)()

	loaded, ok := loadBucketKey("b1")
	require.True(t, ok)
	assert.Equal(t, "old-secret", loaded.SecretAccessKey)

	path, err := keysFilePath()
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "old-secret")
}
