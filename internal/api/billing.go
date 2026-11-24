package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

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

// InvoicePDF is a downloaded invoice PDF and the filename the server suggests.
type InvoicePDF struct {
	Filename string
	Content  []byte
}

// DownloadInvoicePDF downloads an invoice as a raw PDF. Errors are still JSON
// envelopes, so non-200 responses are decoded like every other endpoint.
func (c *Client) DownloadInvoicePDF(ctx context.Context, uid string) (InvoicePDF, error) {
	var result InvoicePDF
	if c.err != nil {
		return result, c.err
	}
	req := c.r.R().SetContext(ctx).SetDoNotParseResponse(true)
	resp, err := req.Get(invoicePath(uid) + "/pdf")
	if err != nil {
		return result, TransportError(err)
	}
	if resp.RawResponse == nil {
		return result, fmt.Errorf("empty response downloading invoice %s", uid)
	}
	defer resp.RawResponse.Body.Close()

	body, err := io.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return result, fmt.Errorf("read invoice %s: %w", uid, err)
	}
	if resp.RawResponse.StatusCode != http.StatusOK {
		var env envelope
		if jsonErr := json.Unmarshal(body, &env); jsonErr == nil && strings.TrimSpace(env.Message) != "" {
			return result, &Error{
				StatusCode: resp.RawResponse.StatusCode,
				Message:    env.Message,
				Action:     env.Action,
			}
		}
		return result, &Error{
			StatusCode: resp.RawResponse.StatusCode,
			Message:    resp.RawResponse.Status,
		}
	}
	result.Content = body
	if cd := resp.RawResponse.Header.Get("Content-Disposition"); cd != "" {
		if _, params, perr := mime.ParseMediaType(cd); perr == nil {
			result.Filename = params["filename"]
		}
	}
	return result, nil
}

func invoicePath(uid string) string {
	return "/v1/invoices/" + url.PathEscape(uid)
}
