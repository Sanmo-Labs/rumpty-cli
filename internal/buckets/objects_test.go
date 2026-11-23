package buckets_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/buckets"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newObjectsRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testBucket() api.AssetBucket {
	return api.AssetBucket{UID: "bkt-1", Name: "assets", Slug: "assets", Status: "ready"}
}

func testObject() api.AssetObject {
	return api.AssetObject{
		UID: "obj-1", BucketUID: "bkt-1", Key: "reports/2026/report.pdf",
		Filename: "report.pdf", ContentType: "application/pdf", SizeBytes: 248331, Status: "ready",
	}
}

func TestObjects_listsKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{testObject()}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Objects(context.Background(), rt, "assets", "", term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "reports/2026/report.pdf")
	assert.Contains(t, s, "242.5 KiB")
}

func TestObjects_prefixFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	other := testObject()
	other.UID, other.Key = "obj-2", "media/hero.webp"
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{testObject(), other}, nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.Objects(context.Background(), rt, "assets", "reports/", term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "reports/2026/report.pdf")
	assert.NotContains(t, s, "media/hero.webp")
}

func TestObjects_bucketNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.Objects(context.Background(), rt, "nope", "", term.OutputTable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket")
}

func TestRmObject_withYesDeletesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{testObject()}, nil)
	mock.EXPECT().DeleteBucketObject(gomock.Any(), "prod-team", "obj-1").Return(nil)

	rt, out := newObjectsRuntime(t, mock)

	require.NoError(t, buckets.RmObject(context.Background(), rt, "assets/reports/2026/report.pdf", true))
	assert.Contains(t, out.String(), "Deleted reports/2026/report.pdf")
}

func TestRmObject_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{testObject()}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.RmObject(context.Background(), rt, "assets/reports/2026/report.pdf", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestRmObject_invalidRef(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.RmObject(context.Background(), rt, "no-slash-here", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "<bucket>/<key>")
}

func TestRmObject_objectNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.RmObject(context.Background(), rt, "assets/missing.pdf", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no object")
}

func TestCp_downloadsThroughReadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello bucket")
	}))
	defer srv.Close()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	obj := api.AssetObject{UID: "obj-1", Key: "reports/2026/report.pdf", SizeBytes: 12, Status: "ready"}
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{obj}, nil)
	mock.EXPECT().CreateObjectReadURL(gomock.Any(), "prod-team", "obj-1").Return(api.ObjectReadURLResult{
		ReadURL:   srv.URL + "/file",
		IsSigned:  true,
		ExpiresAt: "2026-11-23T05:55:00Z",
	}, nil)

	rt, out := newObjectsRuntime(t, mock)
	path := filepath.Join(t.TempDir(), "report.pdf")

	require.NoError(t, buckets.Cp(context.Background(), rt, "assets/reports/2026/report.pdf", path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello bucket"), data)
	assert.Contains(t, out.String(), "Downloaded reports/2026/report.pdf (12 B) to "+path)
}

func TestCp_unknownKeyFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAssetBuckets(gomock.Any(), "prod-team").Return([]api.AssetBucket{testBucket()}, nil)
	mock.EXPECT().ListBucketObjects(gomock.Any(), "prod-team", "bkt-1").Return([]api.AssetObject{}, nil)

	rt, _ := newObjectsRuntime(t, mock)

	err := buckets.Cp(context.Background(), rt, "assets/missing.pdf", "/tmp/x.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no object")
}
