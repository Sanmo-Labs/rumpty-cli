package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
)

type Client struct {
	r *resty.Client
}

func NewClient(baseURL, token string) *Client {
	r := resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetTimeout(30*time.Second).
		SetHeader("Content-Type", "application/json")
	if token != "" {
		r.SetAuthScheme("Bearer").SetAuthToken(token)
	}
	return &Client{r: r}
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

func (c *Client) deleteWithOptions(ctx context.Context, path string, out any, opts requestOptions) error {
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
