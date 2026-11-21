package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

const keysFileName = "s3-keys.json"

type BucketKey struct {
	UID             string `json:"uid,omitempty"`
	Workspace       string `json:"workspace,omitempty"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

func loadBucketKey(bucketUID string) (BucketKey, bool) {
	keys, err := loadKeysFile()
	if err != nil {
		return BucketKey{}, false
	}
	key, ok := keys[bucketUID]
	if !ok || key.AccessKeyID == "" || key.SecretAccessKey == "" || key.Endpoint == "" || key.Bucket == "" {
		return BucketKey{}, false
	}
	return key, true
}

func saveBucketKey(bucketUID string, key *BucketKey) error {
	keys, err := loadKeysFile()
	if err != nil {
		return err
	}
	keys[bucketUID] = *key
	return writeKeysFile(keys)
}

func dropBucketKey(bucketUID string) {
	keys, err := loadKeysFile()
	if err != nil {
		return
	}
	delete(keys, bucketUID)
	_ = writeKeysFile(keys)
}

func keysFilePath() (string, error) {
	dir, err := credentials.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, keysFileName), nil
}

func loadKeysFile() (map[string]BucketKey, error) {
	path, err := keysFilePath()
	if err != nil {
		return nil, err
	}
	credentials.RestrictPermissions(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]BucketKey{}, nil
		}
		return nil, fmt.Errorf("read bucket keys: %w", err)
	}
	keys := map[string]BucketKey{}
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("parse bucket keys: %w", err)
	}
	return keys, nil
}

func writeKeysFile(keys map[string]BucketKey) error {
	path, err := keysFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := credentials.WriteSecretFile(path, data); err != nil {
		return fmt.Errorf("write bucket keys: %w", err)
	}
	return nil
}

// RevokeSavedKeys revokes the bucket access keys sync created and deletes
// the local copies. It returns a description of each key that could not be
// revoked, so the caller can tell the user to revoke it in the console.
func RevokeSavedKeys(ctx context.Context, client api.ClientAPI) ([]string, error) {
	keys, err := loadKeysFile()
	if err != nil {
		return nil, err
	}
	var notRevoked []string
	for _, key := range keys {
		if client != nil && key.UID != "" && key.Workspace != "" {
			if err := client.RevokeAssetAccessKey(ctx, key.Workspace, key.UID); err == nil {
				continue
			}
		}
		notRevoked = append(notRevoked, fmt.Sprintf("access key %s for bucket %s", key.AccessKeyID, key.Bucket))
	}
	sort.Strings(notRevoked)

	path, err := keysFilePath()
	if err != nil {
		return notRevoked, err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return notRevoked, fmt.Errorf("remove bucket keys: %w", err)
	}
	return notRevoked, nil
}
