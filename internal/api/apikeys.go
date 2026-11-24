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
