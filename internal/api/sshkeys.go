package api

import (
	"context"
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
