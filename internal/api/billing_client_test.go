package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadInvoicePDF_returnsPDFAndFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/invoices/inv-1/pdf", r.URL.Path)
		w.Header().Set("Content-Disposition", `attachment; filename="invoice-INV-2026-000123.pdf"`)
		_, _ = w.Write([]byte("%PDF-1.4 test"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok")
	pdf, err := client.DownloadInvoicePDF(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, "invoice-INV-2026-000123.pdf", pdf.Filename)
	assert.Equal(t, "%PDF-1.4 test", string(pdf.Content))
}

func TestDownloadInvoicePDF_decodesJSONErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"message":"invoice not found"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok")
	_, err := client.DownloadInvoicePDF(context.Background(), "inv-1")
	require.Error(t, err)
	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Equal(t, "invoice not found", apiErr.Message)
}
