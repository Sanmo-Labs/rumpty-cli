package api

import (
	"context"
	"net/url"
)

type AssetBucket struct {
	UID              string `json:"uid"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Status           string `json:"status"`
	Visibility       string `json:"visibility"`
	S3BucketName     string `json:"s3_bucket_name"`
	StorageUsedBytes int64  `json:"storage_used_bytes,omitempty"`
	CreatedAt        string `json:"created_at"`
	LastFailure      string `json:"last_failure,omitempty"`
	KeyPrefix        string `json:"key_prefix,omitempty"`
}

type AssetBucketOperationResult struct {
	OperationID string      `json:"operation_id"`
	Bucket      AssetBucket `json:"bucket"`
	Status      string      `json:"status"`
	EventsURL   string      `json:"events_url"`
}

type CreateAssetBucketRequest struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility,omitempty"`
}

type AssetAccessKey struct {
	UID         string   `json:"uid"`
	BucketUID   string   `json:"bucket_uid"`
	Name        string   `json:"name"`
	AccessKeyID string   `json:"access_key_id"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
	Endpoint    string   `json:"endpoint"`
	Region      string   `json:"region"`
	Bucket      string   `json:"bucket"`
	CreatedAt   string   `json:"created_at"`
}

type CreateAssetAccessKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
}

type CreateAssetAccessKeyResult struct {
	AssetAccessKey
	SecretAccessKey string `json:"secret_access_key"`
	OperationID     string `json:"operation_id"`
	OperationStatus string `json:"operation_status"`
}

func (c *Client) ListAssetBuckets(ctx context.Context, workspace string) ([]AssetBucket, error) {
	var data []AssetBucket
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	if err := c.getWithOptions(ctx, "/v1/asset-buckets", &data, opts); err != nil {
		return nil, err
	}
	if data == nil {
		return []AssetBucket{}, nil
	}
	return data, nil
}

func (c *Client) CreateAssetBucket(ctx context.Context, workspace string, req CreateAssetBucketRequest, idempotency string) (AssetBucketOperationResult, error) {
	var data AssetBucketOperationResult
	err := c.post(ctx, "/v1/asset-buckets", req, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) CreateAssetAccessKey(ctx context.Context, workspace, bucketUID string, req CreateAssetAccessKeyRequest) (CreateAssetAccessKeyResult, error) {
	var data CreateAssetAccessKeyResult
	path := "/v1/asset-buckets/" + url.PathEscape(bucketUID) + "/access-keys"
	err := c.post(ctx, path, req, &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	})
	return data, err
}
