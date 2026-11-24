package buckets_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/buckets"
)

func uploadTestFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("hello upload"), 0o600))
	return path
}

func TestUpload_fullFlow(t *testing.T) {
	var gotPut bool
	var gotHeaders http.Header
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPut = r.Method == http.MethodPut
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer storage.Close()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().CreateAssetUpload(gomock.Any(), "prod-team", "bkt-1", &api.CreateAssetUploadRequest{
		Filename:    "report.pdf",
		ContentType: "application/pdf",
		SizeBytes:   12,
		KeyPrefix:   "reports",
	}).Return(api.AssetUploadResult{
		ObjectID: "obj-9", Key: "reports/report-abc.pdf",
		UploadURL: storage.URL + "/put", Method: "PUT",
		Headers: map[string]string{"Content-Type": "application/pdf"},
	}, nil)
	mock.EXPECT().CompleteAssetUpload(gomock.Any(), "prod-team", "obj-9").Return(api.AssetObject{
		UID: "obj-9", BucketUID: "bkt-1", Key: "reports/report-abc.pdf",
		Filename: "report.pdf", ContentType: "application/pdf", SizeBytes: 12, Status: "ready",
	}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Upload(context.Background(), rt, uploadTestFile(t), "assets/reports/report.pdf"))
	assert.True(t, gotPut)
	assert.Equal(t, "application/pdf", gotHeaders.Get("Content-Type"))
	assert.Contains(t, out.String(), "Uploaded")
	assert.Contains(t, out.String(), "reports/report-abc.pdf")
}

func TestUpload_putFailure(t *testing.T) {
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer storage.Close()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().CreateAssetUpload(gomock.Any(), "prod-team", "bkt-1", gomock.Any()).Return(api.AssetUploadResult{
		ObjectID: "obj-9", UploadURL: storage.URL + "/put", Method: "PUT",
		Headers: map[string]string{"Content-Type": "application/pdf"},
	}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.Upload(context.Background(), rt, uploadTestFile(t), "assets/report.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload failed")
}

func TestUpload_missingFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.Upload(context.Background(), rt, "/nonexistent/report.pdf", "assets/report.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
	_ = fmt.Sprint
}
