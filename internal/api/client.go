package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	rumptylog "github.com/cloverworks-hq/rumpty-cli/internal/log"
	"github.com/cloverworks-hq/rumpty-cli/internal/version"
)

type Client struct {
	r   *resty.Client
	err error
}

func NewClient(baseURL, token string) *Client {
	r := resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetTimeout(30*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", UserAgent())
	if token != "" {
		r.SetAuthScheme("Bearer").SetAuthToken(token)
	}
	return &Client{r: r, err: CheckBaseURL(baseURL)}
}

// CheckBaseURL requires HTTPS so API keys are never sent in clear text.
// Plain HTTP is allowed only for loopback addresses, for local development.
func CheckBaseURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid API URL %q", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopback(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("refusing to send credentials over plain HTTP to %s; use an https:// API URL", u.Host)
	default:
		return fmt.Errorf("unsupported API URL scheme %q; use https://", u.Scheme)
	}
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func UserAgent() string {
	return fmt.Sprintf("rumpty-cli/%s (%s; %s)", version.Version, runtime.GOOS, runtime.GOARCH)
}

type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

type requestOptions struct {
	headers map[string]string
}

func (c *Client) post(ctx context.Context, path string, body any, out any, opts requestOptions) error {
	if c.err != nil {
		return c.err
	}
	req := c.r.R().SetContext(ctx)
	if body != nil {
		req.SetBody(body)
	}
	for k, v := range opts.headers {
		req.SetHeader(k, v)
	}
	rumptylog.Debug("API request", "method", "POST", "path", path)
	resp, err := req.Post(path)
	if err != nil {
		return TransportError(err)
	}
	rumptylog.Debug("API response", "method", "POST", "path", path, "status", resp.StatusCode())
	return decodeEnvelope(resp, out)
}

func (c *Client) patch(ctx context.Context, path string, body any, out any, opts requestOptions) error {
	if c.err != nil {
		return c.err
	}
	req := c.r.R().SetContext(ctx).SetBody(body)
	for k, v := range opts.headers {
		req.SetHeader(k, v)
	}
	rumptylog.Debug("API request", "method", "PATCH", "path", path)
	resp, err := req.Patch(path)
	if err != nil {
		return TransportError(err)
	}
	rumptylog.Debug("API response", "method", "PATCH", "path", path, "status", resp.StatusCode())
	return decodeEnvelope(resp, out)
}

func (c *Client) deleteWithOptions(ctx context.Context, path string, out any, opts requestOptions) error {
	if c.err != nil {
		return c.err
	}
	req := c.r.R().SetContext(ctx)
	for k, v := range opts.headers {
		req.SetHeader(k, v)
	}
	rumptylog.Debug("API request", "method", "DELETE", "path", path)
	resp, err := req.Delete(path)
	if err != nil {
		return TransportError(err)
	}
	rumptylog.Debug("API response", "method", "DELETE", "path", path, "status", resp.StatusCode())
	return decodeEnvelope(resp, out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.getWithOptions(ctx, path, out, requestOptions{})
}

func (c *Client) getWithOptions(ctx context.Context, path string, out any, opts requestOptions) error {
	if c.err != nil {
		return c.err
	}
	req := c.r.R().SetContext(ctx)
	for k, v := range opts.headers {
		req.SetHeader(k, v)
	}
	rumptylog.Debug("API request", "method", "GET", "path", path)
	resp, err := req.Get(path)
	if err != nil {
		return TransportError(err)
	}
	rumptylog.Debug("API response", "method", "GET", "path", path, "status", resp.StatusCode())
	return decodeEnvelope(resp, out)
}

func decodeEnvelope(resp *resty.Response, out any) error {
	var env envelope
	if err := json.Unmarshal(resp.Body(), &env); err != nil {
		if resp.IsError() {
			// Not our API's JSON: an older server without this route, or a
			// proxy error page. Say so instead of surfacing a JSON error.
			return &Error{
				StatusCode: resp.StatusCode(),
				Message:    fmt.Sprintf("the API returned %s without details; it may not support this command yet", resp.Status()),
			}
		}
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.IsError() || !env.Success {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = resp.Status()
		}
		return &Error{
			StatusCode: resp.StatusCode(),
			Message:    msg,
			Action:     env.Action,
		}
	}

	if out == nil {
		return nil
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode response payload: %w", err)
	}
	return nil
}
