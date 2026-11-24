package billing_test

import (
	"bytes"
	"context"
	"io"
	"strings"
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

func TestList_printsInvoices(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{
		{
			UID:             "inv-1",
			Number:          "INV-2026-000123",
			Status:          "paid",
			PeriodStart:     "2026-10-01T00:00:00Z",
			PeriodEnd:       "2026-11-01T00:00:00Z",
			IssuedAt:        "2026-11-01T00:05:12Z",
			Currency:        "USD",
			TotalGrossMinor: 1520,
		},
	}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, billing.List(context.Background(), rt, term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "NUMBER")
	assert.Contains(t, s, "INV-2026-000123")
	assert.Contains(t, s, "paid")
	assert.Contains(t, s, "USD 15.20")
}

func TestList_empty(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, billing.List(context.Background(), rt, term.OutputTable))
	assert.Contains(t, strings.TrimSpace(out.String()), "No invoices yet.")
}

func TestList_json(t *testing.T) {
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListInvoices(gomock.Any()).Return([]api.Invoice{
		{UID: "inv-1", Number: "INV-2026-000123", Status: "paid"},
	}, nil)

	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}

	require.NoError(t, billing.List(context.Background(), rt, term.OutputJSON))
	s := out.String()
	assert.Contains(t, s, `"uid": "inv-1"`)
	assert.Contains(t, s, `"number": "INV-2026-000123"`)
}
