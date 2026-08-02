package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
)

func newS3Client(key *BucketKey) (*minio.Client, error) {
	endpoint := strings.TrimSpace(key.Endpoint)
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid storage endpoint %q", key.Endpoint)
	}
	return minio.New(u.Host, &minio.Options{
		Creds:  miniocreds.NewStaticV4(key.AccessKeyID, key.SecretAccessKey, ""),
		Secure: u.Scheme != "http",
		Region: key.Region,
	})
}

func listRemote(ctx context.Context, client *minio.Client, bucket, prefix string) (map[string]FileInfo, error) {
	listPrefix := prefix
	if listPrefix != "" {
		listPrefix += "/"
	}
	files := make(map[string]FileInfo)
	for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: listPrefix, Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		key := strings.TrimPrefix(obj.Key, listPrefix)
		if key == "" || strings.HasSuffix(key, "/") {
			continue
		}
		files[key] = FileInfo{Key: key, Size: obj.Size, ModTime: obj.LastModified}
	}
	return files, nil
}

func isAuthError(err error) bool {
	var resp minio.ErrorResponse
	if !errors.As(err, &resp) {
		return false
	}
	switch resp.Code {
	case "InvalidAccessKeyId", "SignatureDoesNotMatch", "AccessDenied":
		return true
	}
	return resp.StatusCode == http.StatusForbidden
}
