package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
)

const keysFileName = "s3-keys.json"

type BucketKey struct {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write bucket keys: %w", err)
	}
	return nil
}
