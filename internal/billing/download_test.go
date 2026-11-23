package billing_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/billing"
)

func TestDownloadInvoice_savesPDF(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{testInvoice()}, nil)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1", "INV-2026-0007").Return([]byte("%PDF-1.4 test"), "", nil)

	rt, out := newBillingRuntime(t, mock)
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.pdf")

	require.NoError(t, billing.DownloadInvoice(context.Background(), rt, "INV-2026-0007", path, false))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 test", string(data))
	assert.Contains(t, out.String(), "Saved INV-2026-0007")
}

func TestDownloadInvoice_refusesOverwrite(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{testInvoice()}, nil)

	rt, _ := newBillingRuntime(t, mock)
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.pdf")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	err := billing.DownloadInvoice(context.Background(), rt, "inv-1", path, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))
}

func TestDownloadInvoice_forceOverwrites(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{testInvoice()}, nil)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1", "INV-2026-0007").Return([]byte("%PDF-1.4 new"), "invoice-INV-2026-0007.pdf", nil)

	rt, out := newBillingRuntime(t, mock)
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.pdf")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	require.NoError(t, billing.DownloadInvoice(context.Background(), rt, "inv-1", path, true))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 new", string(data))
	assert.Contains(t, out.String(), "INV-2026-0007")
}

func TestDownloadInvoice_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{}, nil)

	rt, _ := newBillingRuntime(t, mock)

	err := billing.DownloadInvoice(context.Background(), rt, "INV-9999", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
