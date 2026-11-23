package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// InvoiceCompany is the company issuing an invoice.
type InvoiceCompany struct {
	Name    string `json:"name"`
	TaxID   string `json:"tax_id,omitempty"`
	Address string `json:"address,omitempty"`
}

// InvoiceLine is one billed item on an invoice.
type InvoiceLine struct {
	Description  string  `json:"description"`
	ResourceType string  `json:"resource_type,omitempty"`
	Metric       string  `json:"metric,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	Quantity     float64 `json:"quantity"`
	NetCents     int64   `json:"net_cents"`
	TaxCents     int64   `json:"tax_cents"`
	TaxRateBps   int64   `json:"tax_rate_bps"`
	TaxType      string  `json:"tax_type"`
	TaxReason    string  `json:"tax_reason"`
}

// Invoice is one billing invoice with its line items.
type Invoice struct {
	UID              string         `json:"uid"`
	Number           string         `json:"number"`
	Status           string         `json:"status"`
	PeriodStart      string         `json:"period_start"`
	PeriodEnd        string         `json:"period_end"`
	IssuedAt         string         `json:"issued_at,omitempty"`
	PaidAt           string         `json:"paid_at,omitempty"`
	BaseCurrency     string         `json:"base_currency"`
	SubtotalNetCents int64          `json:"subtotal_net_cents"`
	TaxTotalCents    int64          `json:"tax_total_cents"`
	TotalGrossCents  int64          `json:"total_gross_cents"`
	Currency         string         `json:"currency"`
	ExchangeRate     float64        `json:"exchange_rate"`
	SubtotalNetMinor int64          `json:"subtotal_net_minor"`
	TaxTotalMinor    int64          `json:"tax_total_minor"`
	TotalGrossMinor  int64          `json:"total_gross_minor"`
	TaxType          string         `json:"tax_type"`
	TaxRateBps       int64          `json:"tax_rate_bps"`
	TaxReason        string         `json:"tax_reason"`
	CustomerName     string         `json:"customer_name,omitempty"`
	CustomerEmail    string         `json:"customer_email,omitempty"`
	CustomerCountry  string         `json:"customer_country,omitempty"`
	CustomerTaxID    string         `json:"customer_tax_id,omitempty"`
	Company          InvoiceCompany `json:"company"`
	Lines            []InvoiceLine  `json:"lines"`
}

// ListInvoices returns your 50 most recent invoices, newest first.
func (c *Client) ListInvoices(ctx context.Context) ([]Invoice, error) {
	var data []Invoice
	if err := c.get(ctx, "/v1/invoices", &data); err != nil {
		return nil, err
	}
	if data == nil {
		return []Invoice{}, nil
	}
	return data, nil
}

// DownloadInvoicePDF fetches an invoice's PDF. The filename comes from the
// Content-Disposition header, falling back to invoice-<number>.pdf.
func (c *Client) DownloadInvoicePDF(ctx context.Context, uid, number string) ([]byte, string, error) {
	if c.err != nil {
		return nil, "", c.err
	}
	resp, err := c.r.R().SetContext(ctx).Get("/v1/invoices/" + strings.ReplaceAll(uid, "/", "") + "/pdf")
	if err != nil {
		return nil, "", TransportError(err)
	}
	if resp.StatusCode() >= 400 {
		var env envelope
		if jerr := json.Unmarshal(resp.Body(), &env); jerr == nil && env.Message != "" {
			return nil, "", fmt.Errorf("%s", env.Message)
		}
		return nil, "", fmt.Errorf("invoice download failed: %s", resp.Status())
	}
	return resp.Body(), invoiceFilename(resp.Header().Get("Content-Disposition"), number), nil
}

// invoiceFilename extracts the filename from a Content-Disposition header.
func invoiceFilename(disposition, number string) string {
	for _, part := range strings.Split(disposition, ";") {
		part = strings.TrimSpace(part)
		if name, ok := strings.CutPrefix(part, "filename="); ok {
			name = strings.Trim(name, "\"")
			if name != "" {
				return name
			}
		}
	}
	return fmt.Sprintf("invoice-%s.pdf", strings.ReplaceAll(number, "/", "-"))
}
