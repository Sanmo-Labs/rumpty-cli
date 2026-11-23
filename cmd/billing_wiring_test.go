package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// The help text and docs point at `rumpty billing invoices download <id>`;
// this keeps the documented command path working.
func TestBillingInvoicesDownload_documentedCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().DownloadInvoicePDF(gomock.Any(), "inv-1").Return(api.InvoicePDF{
		Filename: "invoice-INV-2026-0001.pdf",
		Content:  []byte("%PDF-1.4 test"),
	}, nil)

	t.Chdir(t.TempDir())
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: &bytes.Buffer{}, ErrOut: io.Discard},
		APIClient: mock,
	}

	cmd := newBillingCmd(rt)
	cmd.SetArgs([]string{"invoices", "download", "inv-1"})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile("invoice-INV-2026-0001.pdf")
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4 test", string(data))
}
