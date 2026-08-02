package sync

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

const (
	bucketPollInterval = 2 * time.Second
	bucketPollTimeout  = 5 * time.Minute
	keyPollInterval    = 2 * time.Second
	keyPollTimeout     = 3 * time.Minute
)

func ensureBucket(ctx context.Context, rt *app.Runtime, workspace, name string, allowCreate, public bool) (api.AssetBucket, error) {
	visibility := "private"
	if public {
		visibility = "public"
	}

	bucket, found, err := findBucket(ctx, rt, workspace, name)
	if err != nil {
		return api.AssetBucket{}, err
	}
	if found {
		if public && bucket.Visibility != "public" {
			term.Statusf(rt.Streams.ErrOut, "Bucket %s already exists as %s; --public only applies when creating a bucket", bucket.Name, bucket.Visibility)
		}
		return waitForBucketReady(ctx, rt, workspace, &bucket)
	}

	if !allowCreate {
		return api.AssetBucket{}, fmt.Errorf("bucket %q not found in workspace %s", name, workspace)
	}
	term.Statusf(rt.Streams.ErrOut, "Creating %s bucket %s", visibility, name)
	result, err := rt.API().CreateAssetBucket(ctx, workspace, api.CreateAssetBucketRequest{
		Name:       name,
		Visibility: visibility,
	}, fmt.Sprintf("cli-sync-bucket-%s-%d", name, time.Now().UnixNano()))
	if err != nil {
		return api.AssetBucket{}, err
	}
	return waitForBucketReady(ctx, rt, workspace, &result.Bucket)
}

func findBucket(ctx context.Context, rt *app.Runtime, workspace, name string) (api.AssetBucket, bool, error) {
	buckets, err := rt.API().ListAssetBuckets(ctx, workspace)
	if err != nil {
		return api.AssetBucket{}, false, err
	}
	for i := range buckets {
		if strings.EqualFold(buckets[i].Name, name) || strings.EqualFold(buckets[i].Slug, name) {
			return buckets[i], true, nil
		}
	}
	return api.AssetBucket{}, false, nil
}

func waitForBucketReady(ctx context.Context, rt *app.Runtime, workspace string, bucket *api.AssetBucket) (api.AssetBucket, error) {
	if bucket.Status == "ready" {
		return *bucket, nil
	}
	if bucket.Status != "provisioning" && bucket.Status != "" {
		return api.AssetBucket{}, bucketFailure(bucket)
	}

	spin := term.StartSpinner(rt.Streams.ErrOut, fmt.Sprintf("Waiting for bucket %s to become ready", bucket.Name))
	defer spin.Stop()

	deadline := time.Now().Add(bucketPollTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return api.AssetBucket{}, err
		}
		if time.Now().After(deadline) {
			return api.AssetBucket{}, fmt.Errorf("timed out waiting for bucket %s to become ready", bucket.Name)
		}
		current, found, err := findBucket(ctx, rt, workspace, bucket.Name)
		if err != nil {
			return api.AssetBucket{}, err
		}
		if found {
			switch current.Status {
			case "ready":
				return current, nil
			case "provisioning":
			default:
				return api.AssetBucket{}, bucketFailure(&current)
			}
		}
		select {
		case <-ctx.Done():
			return api.AssetBucket{}, ctx.Err()
		case <-time.After(bucketPollInterval):
		}
	}
}

func bucketFailure(bucket *api.AssetBucket) error {
	if strings.TrimSpace(bucket.LastFailure) != "" {
		return fmt.Errorf("bucket %s is %s: %s", bucket.Name, bucket.Status, bucket.LastFailure)
	}
	return fmt.Errorf("bucket %s is %s", bucket.Name, bucket.Status)
}

func ensureBucketKey(ctx context.Context, rt *app.Runtime, workspace string, bucket *api.AssetBucket) (BucketKey, bool, error) {
	if key, ok := loadBucketKey(bucket.UID); ok {
		return key, true, nil
	}
	key, err := mintBucketKey(ctx, rt, workspace, bucket)
	return key, false, err
}

func mintBucketKey(ctx context.Context, rt *app.Runtime, workspace string, bucket *api.AssetBucket) (BucketKey, error) {
	term.Statusf(rt.Streams.ErrOut, "Creating access key for bucket %s", bucket.Name)
	result, err := rt.API().CreateAssetAccessKey(ctx, workspace, bucket.UID, api.CreateAssetAccessKeyRequest{
		Name: accessKeyName(),
	})
	if err != nil {
		return BucketKey{}, err
	}
	if err := waitForOperation(ctx, rt, workspace, result.OperationID, "access key"); err != nil {
		return BucketKey{}, err
	}
	key := BucketKey{
		AccessKeyID:     result.AccessKeyID,
		SecretAccessKey: result.SecretAccessKey,
		Endpoint:        result.Endpoint,
		Region:          result.Region,
		Bucket:          result.Bucket,
	}
	if key.Bucket == "" {
		key.Bucket = bucket.S3BucketName
	}
	if err := saveBucketKey(bucket.UID, &key); err != nil {
		return BucketKey{}, err
	}
	return key, nil
}

func accessKeyName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown"
	}
	host = strings.ToLower(strings.Split(host, ".")[0])
	return "rumpty-sync-" + host
}

func waitForOperation(ctx context.Context, rt *app.Runtime, workspace, operationID, what string) error {
	if strings.TrimSpace(operationID) == "" {
		return nil
	}
	deadline := time.Now().Add(keyPollTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to become ready", what)
		}
		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("creating %s failed", what)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(keyPollInterval):
		}
	}
}
