package buckets_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/buckets"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func testKey() api.AssetAccessKey {
	return api.AssetAccessKey{
		UID: "key-1", BucketUID: "bkt-1", Name: "deploy", AccessKeyID: "RPTY123",
		Permissions: []string{"read", "list"}, Status: "active",
		Endpoint: "https://s3.rumptycloud.com", Region: "us-east-1", Bucket: "assets",
	}
}

func TestKeys_listsKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListAssetAccessKeys(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetAccessKey{testKey()}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Keys(context.Background(), rt, "assets", term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "deploy")
	assert.Contains(t, s, "RPTY123")
	assert.Contains(t, s, "read,list")
}

func TestKeys_emptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListAssetAccessKeys(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetAccessKey{}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Keys(context.Background(), rt, "assets", term.OutputTable))
	assert.Contains(t, out.String(), "No access keys yet")
}

func TestKeys_jsonOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListAssetAccessKeys(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetAccessKey{testKey()}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Keys(context.Background(), rt, "assets", term.OutputJSON))
	assert.Contains(t, out.String(), "access_key_id")
	assert.Contains(t, out.String(), "RPTY123")
}

func TestCreateKey_printsSecretOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().CreateAssetAccessKey(gomock.Any(), "prod-team", "bkt-1", api.CreateAssetAccessKeyRequest{
		Name:        "deploy",
		Permissions: []string{"read", "list"},
	}).Return(api.CreateAssetAccessKeyResult{
		AssetAccessKey:  testKey(),
		SecretAccessKey: "SHHH",
		OperationID:     "op-1",
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{UID: "op-1", Status: "succeeded"}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.CreateKey(context.Background(), rt, "assets", "deploy", []string{"read", "list"}))
	s := out.String()
	assert.Contains(t, s, "secret_access_key: SHHH")
	assert.Contains(t, s, "The key is active")
}

func TestCreateKey_operationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().CreateAssetAccessKey(gomock.Any(), "prod-team", "bkt-1", gomock.Any()).Return(api.CreateAssetAccessKeyResult{
		AssetAccessKey:  testKey(),
		SecretAccessKey: "SHHH",
		OperationID:     "op-2",
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-2").Return(api.Operation{UID: "op-2", Status: "failed"}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.CreateKey(context.Background(), rt, "assets", "deploy", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provisioning failed")
}

func TestRevokeKey_renames(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListAssetAccessKeys(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetAccessKey{testKey()}, nil)
	mock.EXPECT().RevokeAssetAccessKey(gomock.Any(), "prod-team", "key-1").Return(nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.RevokeKey(context.Background(), rt, "assets", "RPTY123", true))
	assert.Contains(t, out.String(), "Revoked access key deploy")
}

func TestRevokeKey_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListAssetAccessKeys(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetAccessKey{testKey()}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.RevokeKey(context.Background(), rt, "assets", "missing", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
