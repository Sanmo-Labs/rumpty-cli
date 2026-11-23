package api

import (
	"context"
	"net/url"
)

// SSHKey is an SSH public key registered on the account.
type SSHKey struct {
	UID         string `json:"uid"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// AddSSHKeyRequest is the body for adding an SSH public key.
type AddSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// ListSSHKeys returns the SSH public keys visible in a workspace.
func (c *Client) ListSSHKeys(ctx context.Context, workspace string) ([]SSHKey, error) {
	var data []SSHKey
	if err := c.getWithOptions(ctx, "/v1/ssh-keys", &data, workspaceRequestOptions(workspace, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []SSHKey{}, nil
	}
	return data, nil
}

// DeleteSSHKey removes an SSH key from the account.
func (c *Client) DeleteSSHKey(ctx context.Context, workspace, uid string) error {
	return c.deleteWithOptions(ctx, "/v1/ssh-keys/"+url.PathEscape(uid), nil, workspaceRequestOptions(workspace, ""))
}

// AddSSHKey adds an SSH public key to the account and the workspace.
func (c *Client) AddSSHKey(ctx context.Context, workspace string, req AddSSHKeyRequest) (SSHKey, error) {
	var data SSHKey
	err := c.post(ctx, "/v1/ssh-keys", req, &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	})
	return data, err
}
