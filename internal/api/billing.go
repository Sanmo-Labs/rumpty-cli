package api

import "context"

// InvoiceCompany describes the issuer of an invoice.
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
	NetCents     int     `json:"net_cents"`
	TaxCents     int     `json:"tax_cents"`
	TaxRateBps   int     `json:"tax_rate_bps"`
	TaxType      string  `json:"tax_type"`
	TaxReason    string  `json:"tax_reason"`
}

// Invoice is a billing invoice with its line items.
type Invoice struct {
	UID              string         `json:"uid"`
	Number           string         `json:"number"`
	Status           string         `json:"status"`
	PeriodStart      string         `json:"period_start"`
	PeriodEnd        string         `json:"period_end"`
	IssuedAt         string         `json:"issued_at,omitempty"`
	PaidAt           string         `json:"paid_at,omitempty"`
	BaseCurrency     string         `json:"base_currency"`
	SubtotalNetCents int            `json:"subtotal_net_cents"`
	TaxTotalCents    int            `json:"tax_total_cents"`
	TotalGrossCents  int            `json:"total_gross_cents"`
	Currency         string         `json:"currency"`
	ExchangeRate     float64        `json:"exchange_rate"`
	SubtotalNetMinor int            `json:"subtotal_net_minor"`
	TaxTotalMinor    int            `json:"tax_total_minor"`
	TotalGrossMinor  int            `json:"total_gross_minor"`
	TaxType          string         `json:"tax_type"`
	TaxRateBps       int            `json:"tax_rate_bps"`
	TaxReason        string         `json:"tax_reason"`
	CustomerName     string         `json:"customer_name,omitempty"`
	CustomerEmail    string         `json:"customer_email,omitempty"`
	CustomerCountry  string         `json:"customer_country,omitempty"`
	CustomerTaxID    string         `json:"customer_tax_id,omitempty"`
	Company          InvoiceCompany `json:"company"`
	Lines            []InvoiceLine  `json:"lines"`
}

// ListInvoices returns the account's most recent invoices, newest first.
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
