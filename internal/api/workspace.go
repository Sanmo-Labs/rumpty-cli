package api

import "context"

func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var data []Workspace
	if err := c.get(ctx, "/v1/workspaces", &data); err != nil {
		return nil, err
	}
	if data == nil {
		return []Workspace{}, nil
	}
	return data, nil
}
