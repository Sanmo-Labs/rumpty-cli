package billing_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/billing"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newDownloadRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func TestDownload_savesServerSuggestedName(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1").Return(api.InvoicePDF{
		Filename: "invoice-INV-2026-000123.pdf",
		Content:  []byte("%PDF-1.4 test"),
	}, nil)

	t.Chdir(t.TempDir())
	rt, out := newDownloadRuntime(t, mock)

	require.NoError(t, billing.Download(context.Background(), rt, "inv-1", "", false))
	data, err := os.ReadFile("invoice-INV-2026-000123.pdf")
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 test", string(data))
	assert.Contains(t, out.String(), "Saved invoice to invoice-INV-2026-000123.pdf")
}

func TestDownload_writesToOutputPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1").Return(api.InvoicePDF{
		Content: []byte("%PDF-1.4 nested"),
	}, nil)

	t.Chdir(t.TempDir())
	rt, _ := newDownloadRuntime(t, mock)

	require.NoError(t, billing.Download(context.Background(), rt, "inv-1", filepath.Join("downloads", "nov.pdf"), false))
	data, err := os.ReadFile(filepath.Join("downloads", "nov.pdf"))
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 nested", string(data))
}

func TestDownload_refusesOverwriteWithoutForce(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1").Return(api.InvoicePDF{
		Filename: "invoice.pdf",
		Content:  []byte("new"),
	}, nil)

	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("invoice.pdf", []byte("old"), 0o600))
	rt, _ := newDownloadRuntime(t, mock)

	err := billing.Download(context.Background(), rt, "inv-1", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile("invoice.pdf")
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(data))
}

func TestDownload_forceOverwrites(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1").Return(api.InvoicePDF{
		Filename: "invoice.pdf",
		Content:  []byte("new"),
	}, nil)

	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("invoice.pdf", []byte("old"), 0o600))
	rt, _ := newDownloadRuntime(t, mock)

	require.NoError(t, billing.Download(context.Background(), rt, "inv-1", "", true))
	data, err := os.ReadFile("invoice.pdf")
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}
