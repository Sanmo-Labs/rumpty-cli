package api

import (
	"context"
	"fmt"
)

func (c *Client) IssueSSHCert(ctx context.Context, workspace string, req CertRequest) (CertResponse, error) {
	var data CertResponse
	opts := requestOptions{
		headers: map[string]string{
			"X-Workspace-Slug": workspace,
		},
	}
	if err := c.post(ctx, "/v1/ssh-sessions/cert", req, &data, opts); err != nil {
		return CertResponse{}, err
	}
	if data.EdgeHost == "" || data.RouterUser == "" || data.Certificate == "" {
		return CertResponse{}, fmt.Errorf("ssh certificate response is incomplete")
	}
	if data.EdgePort == 0 {
		data.EdgePort = 22
	}
	return data, nil
}
