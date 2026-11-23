package api

import (
	"context"
	"net/url"
)

// APIKey is one of the user's personal API keys.
type APIKey struct {
	UID          string   `json:"uid"`
	Name         string   `json:"name"`
	Prefix       string   `json:"prefix"`
	Scopes       []string `json:"scopes"`
	Permissions  []string `json:"permissions"`
	NeverExpires bool     `json:"never_expires,omitempty"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	LastUsedAt   string   `json:"last_used_at,omitempty"`
}

// ListAPIKeys returns the account's active API keys.
func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var data []APIKey
	if err := c.get(ctx, "/v1/api-keys", &data); err != nil {
		return nil, err
	}
	if data == nil {
		return []APIKey{}, nil
	}
	return data, nil
}

// RevokeAPIKey revokes an API key.
func (c *Client) RevokeAPIKey(ctx context.Context, uid string) error {
	return c.deleteWithOptions(ctx, "/v1/api-keys/"+url.PathEscape(uid), nil, requestOptions{})
}

// CreateAPIKeyRequest is the body for creating a personal API key.
type CreateAPIKeyRequest struct {
	Name          string   `json:"name"`
	ExpiresInDays int      `json:"expires_in_days,omitempty"`
	NeverExpires  bool     `json:"never_expires,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
}

// CreateAPIKeyResult is the response for creating an API key. Token is the
// full secret token, shown only once.
type CreateAPIKeyResult struct {
	APIKey
	Token string `json:"token"`
}

// CreateAPIKey creates a personal API key. The full token appears only in
// this response and cannot be retrieved again.
func (c *Client) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (CreateAPIKeyResult, error) {
	var data CreateAPIKeyResult
	err := c.post(ctx, "/v1/api-keys", req, &data, requestOptions{})
	return data, err
}
