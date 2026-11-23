package billing_test

import (
	"bytes"
	"context"
	"io"
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

func newBillingRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testInvoice() api.Invoice {
	return api.Invoice{
		UID: "inv-1", Number: "INV-2026-0007", Status: "paid",
		PeriodStart: "2026-10-01T00:00:00Z", PeriodEnd: "2026-10-31T23:59:59Z",
		IssuedAt: "2026-11-01T10:00:00Z", PaidAt: "2026-11-02T09:00:00Z",
		Currency: "NGN", TotalGrossMinor: 47258,
	}
}

func TestInvoices_lists(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{testInvoice()}, nil)

	rt, out := newBillingRuntime(t, mock)

	require.NoError(t, billing.Invoices(context.Background(), rt, term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "INV-2026-0007")
	assert.Contains(t, s, "paid")
	assert.Contains(t, s, "NGN")
}

func TestInvoices_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{}, nil)

	rt, out := newBillingRuntime(t, mock)

	require.NoError(t, billing.Invoices(context.Background(), rt, term.OutputTable))
	assert.Contains(t, out.String(), "No invoices yet")
}

func TestInvoices_json(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{testInvoice()}, nil)

	rt, out := newBillingRuntime(t, mock)

	require.NoError(t, billing.Invoices(context.Background(), rt, term.OutputJSON))
	assert.Contains(t, out.String(), "INV-2026-0007")
	assert.Contains(t, out.String(), "total_gross_minor")
}
