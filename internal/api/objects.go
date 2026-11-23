package api

import (
	"context"
	"net/url"
)

// AssetObject is an object stored in a bucket.
type AssetObject struct {
	UID              string `json:"uid"`
	BucketUID        string `json:"bucket_uid"`
	Key              string `json:"key"`
	Filename         string `json:"filename"`
	ContentType      string `json:"content_type"`
	SizeBytes        int64  `json:"size_bytes"`
	Status           string `json:"status"`
	PublicURL        string `json:"public_url"`
	ReadURL          string `json:"read_url,omitempty"`
	ReadURLExpiresAt string `json:"read_url_expires_at,omitempty"`
	ReadURLIsSigned  bool   `json:"read_url_is_signed"`
	CreatedAt        string `json:"created_at,omitempty"`
	LastFailure      string `json:"last_failure,omitempty"`
}

// ListBucketObjects returns every object in a bucket, newest first. The
// endpoint is not paginated.
func (c *Client) ListBucketObjects(ctx context.Context, workspace, bucketUID string) ([]AssetObject, error) {
	var data []AssetObject
	if err := c.getWithOptions(ctx, "/v1/asset-buckets/"+url.PathEscape(bucketUID)+"/objects", &data, workspaceRequestOptions(workspace, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []AssetObject{}, nil
	}
	return data, nil
}

// DeleteBucketObject deletes an object immediately. It cannot be undone.
func (c *Client) DeleteBucketObject(ctx context.Context, workspace, objectUID string) error {
	return c.deleteWithOptions(ctx, "/v1/asset-objects/"+url.PathEscape(objectUID), nil, workspaceRequestOptions(workspace, ""))
}

// ObjectReadURLResult is the response for creating an object read URL. For
// a public bucket the permanent public URL is returned, is_signed is false
// and expires_at is omitted.
type ObjectReadURLResult struct {
	ReadURL   string `json:"read_url"`
	ExpiresAt string `json:"expires_at,omitempty"`
	IsSigned  bool   `json:"is_signed"`
}

// CreateObjectReadURL returns a URL to download one ready object. In a
// private bucket the URL is presigned and works until expires_at.
func (c *Client) CreateObjectReadURL(ctx context.Context, workspace, objectUID string) (ObjectReadURLResult, error) {
	var data ObjectReadURLResult
	err := c.post(ctx, "/v1/asset-objects/"+url.PathEscape(objectUID)+"/read-url",
		nil, &data, workspaceRequestOptions(workspace, ""))
	return data, err
}

// CreateAssetUploadRequest is the body for reserving an object upload.
type CreateAssetUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	KeyPrefix   string `json:"key_prefix,omitempty"`
}

// CreateAssetUpload reserves an object and returns a presigned URL to PUT
// the file to. The URL expires at ExpiresAt; send every header exactly as
// returned, then call CompleteAssetUpload.
func (c *Client) CreateAssetUpload(ctx context.Context, workspace, bucketUID string, req *CreateAssetUploadRequest) (AssetUploadResult, error) {
	var data AssetUploadResult
	err := c.post(ctx, "/v1/asset-buckets/"+url.PathEscape(bucketUID)+"/uploads", req, &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	})
	return data, err
}

// CompleteAssetUpload marks an object uploaded with CreateAssetUpload as
// ready. Completing an already-ready object is safe and returns it
// unchanged.
func (c *Client) CompleteAssetUpload(ctx context.Context, workspace, objectUID string) (AssetObject, error) {
	var data AssetObject
	err := c.post(ctx, "/v1/asset-objects/"+url.PathEscape(objectUID)+"/complete", nil, &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	})
	return data, err
}

// AssetUploadResult is the response of Create an upload: a reserved object
// with a presigned URL to send the file's bytes to.
type AssetUploadResult struct {
	ObjectID         string            `json:"object_id"`
	Key              string            `json:"key"`
	UploadURL        string            `json:"upload_url"`
	PublicURL        string            `json:"public_url"`
	ReadURL          string            `json:"read_url,omitempty"`
	ReadURLExpiresAt string            `json:"read_url_expires_at,omitempty"`
	ReadURLIsSigned  bool              `json:"read_url_is_signed"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers"`
	ExpiresAt        string            `json:"expires_at"`
}
