package api

import "context"

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
