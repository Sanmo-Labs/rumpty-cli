package buckets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_keepsDestinationWhenDownloadFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		panic(http.ErrAbortHandler)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "obj.bin")
	require.NoError(t, os.WriteFile(dest, []byte("original"), 0o600))

	_, err := downloadFile(context.Background(), srv.URL, dest)
	require.Error(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

func TestDownloadFile_replacesDestinationOnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new contents"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "obj.bin")
	require.NoError(t, os.WriteFile(dest, []byte("original"), 0o600))

	n, err := downloadFile(context.Background(), srv.URL, dest)
	require.NoError(t, err)
	assert.EqualValues(t, len("new contents"), n)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "new contents", string(data))
}
