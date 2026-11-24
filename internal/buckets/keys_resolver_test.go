package buckets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

func TestFindAccessKey_exactIDBeatsDuplicateNames(t *testing.T) {
	keys := []api.AssetAccessKey{
		{UID: "key-1", Name: "deploy", AccessKeyID: "RUMPTY111"},
		{UID: "key-2", Name: "deploy", AccessKeyID: "RUMPTY222"},
	}

	key, err := findAccessKey(keys, "RUMPTY222", "assets")
	require.NoError(t, err)
	assert.Equal(t, "key-2", key.UID)
}

func TestFindAccessKey_ambiguousNameErrors(t *testing.T) {
	keys := []api.AssetAccessKey{
		{UID: "key-1", Name: "deploy", AccessKeyID: "RUMPTY111"},
		{UID: "key-2", Name: "deploy", AccessKeyID: "RUMPTY222"},
	}

	_, err := findAccessKey(keys, "deploy", "assets")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "key-1")
	assert.Contains(t, err.Error(), "key-2")
}

func TestFindAccessKey_uniqueNameMatches(t *testing.T) {
	keys := []api.AssetAccessKey{
		{UID: "key-1", Name: "deploy", AccessKeyID: "RUMPTY111"},
		{UID: "key-2", Name: "ci", AccessKeyID: "RUMPTY222"},
	}

	key, err := findAccessKey(keys, "CI", "assets")
	require.NoError(t, err)
	assert.Equal(t, "key-2", key.UID)
}
