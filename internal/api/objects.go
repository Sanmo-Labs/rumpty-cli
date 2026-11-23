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
