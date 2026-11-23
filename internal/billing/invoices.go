package billing

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Invoices lists your invoices, newest first.
func Invoices(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	spin := term.StartLoading(errOut(rt), "Listing invoices")
	invoices, err := rt.API().ListInvoices(ctx)
	spin.Stop()
	if err != nil {
		return err
	}

	out := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(out, invoices)
	}
	if len(invoices) == 0 {
		fmt.Fprintln(out, term.Muted(out, "No invoices yet. Invoices are created for completed billing cycles."))
		return nil
	}
	table := term.NewTable(out, "NUMBER", "PERIOD", "STATUS", "TOTAL", "ISSUED")
	for i := range invoices {
		inv := &invoices[i]
		table.Row(inv.Number, fmt.Sprintf("%s to %s", shortDate(inv.PeriodStart), shortDate(inv.PeriodEnd)), inv.Status, formatMoney(inv), shortDate(inv.IssuedAt))
	}
	return table.Render()
}

// shortDate renders a date-time as its date.
func shortDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

// formatMoney renders an invoice total in its charge currency.
func formatMoney(inv *api.Invoice) string {
	return fmt.Sprintf("%s %s", inv.Currency, humanMoney(inv))
}

// humanMoney renders the total as major units.
func humanMoney(inv *api.Invoice) string {
	units := float64(inv.TotalGrossMinor) / 100.0
	if inv.Currency == "NGN" {
		return fmt.Sprintf("₦%.2f", units)
	}
	return fmt.Sprintf("%.2f %s", units, inv.Currency)
}

// errOut returns the error stream so loading spinners stay off stdout.
func errOut(rt *app.Runtime) io.Writer {
	return rt.Streams.ErrOut
}

// DownloadInvoice saves an invoice's PDF to outputPath, refusing to
// overwrite an existing file without force.
func DownloadInvoice(ctx context.Context, rt *app.Runtime, uid, outputPath string, force bool) error {
	w := errOut(rt)

	spin := term.StartLoading(w, "Finding invoice "+uid)
	invoices, err := rt.API().ListInvoices(ctx)
	spin.Stop()
	if err != nil {
		return err
	}
	var invoice *api.Invoice
	for i := range invoices {
		if strings.EqualFold(invoices[i].UID, uid) || strings.EqualFold(invoices[i].Number, uid) {
			invoice = &invoices[i]
			break
		}
	}
	if invoice == nil {
		return fmt.Errorf("invoice %q not found (list your invoices with rumpty billing invoices)", uid)
	}
	if outputPath == "" {
		outputPath = fmt.Sprintf("invoice-%s.pdf", strings.ReplaceAll(invoice.Number, "/", "-"))
	}
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", outputPath)
		}
	}

	spin = term.StartLoading(w, "Downloading "+invoice.Number)
	data, _, err := rt.API().DownloadInvoicePDF(ctx, invoice.UID, invoice.Number)
	spin.Stop()
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Saved %s (%s) to %s", invoice.Number, humanMoney(invoice), outputPath)))
	return nil
}
