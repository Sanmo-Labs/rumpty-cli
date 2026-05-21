package api

import "context"

func (c *Client) ListVMs(ctx context.Context, workspace string) ([]VM, error) {
	var data []VM
	opts := requestOptions{
		headers: map[string]string{
			"X-Workspace-Slug": workspace,
		},
	}
	if err := c.getWithOptions(ctx, "/v1/vms", &data, opts); err != nil {
		return nil, err
	}
	if data == nil {
		return []VM{}, nil
	}
	return data, nil
}
