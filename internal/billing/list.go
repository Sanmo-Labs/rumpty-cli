package billing

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// List prints the account's invoices as a table or JSON.
func List(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	w := rt.Streams.Out
	spin := term.StartLoading(errOut(rt), "Loading invoices")
	invoices, err := rt.API().ListInvoices(ctx)
	spin.Stop()
	if err != nil {
		return err
	}

	if format == term.OutputJSON {
		return term.WriteJSON(w, invoices)
	}
	if len(invoices) == 0 {
		fmt.Fprintln(w, "No invoices yet.")
		return nil
	}

	table := term.NewTable(w, "NUMBER", "STATUS", "PERIOD", "ISSUED", "TOTAL")
	for i := range invoices {
		inv := &invoices[i]
		table.Row(
			term.Bold(w, inv.Number),
			inv.Status,
			period(inv),
			issued(inv),
			formatMoney(inv),
		)
	}
	if err := table.Render(); err != nil {
		return err
	}
	term.Summary(w, plural(len(invoices), "invoice", "invoices"))
	return nil
}

func period(inv *api.Invoice) string {
	return shortDate(inv.PeriodStart) + " to " + shortDate(inv.PeriodEnd)
}

func issued(inv *api.Invoice) string {
	if strings.TrimSpace(inv.IssuedAt) == "" {
		return "-"
	}
	return fullDate(inv.IssuedAt)
}

func formatMoney(inv *api.Invoice) string {
	return fmt.Sprintf("%s %.2f", inv.Currency, float64(inv.TotalGrossMinor)/100)
}

func shortDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format("Jan 2")
}

func fullDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format("Jan 2, 2006")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
