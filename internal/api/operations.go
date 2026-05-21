package api

import (
	"context"
	"net/url"
)

func (c *Client) GetOperation(ctx context.Context, workspace, operationID string) (Operation, error) {
	var data Operation
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	path := "/v1/operations/" + url.PathEscape(operationID)
	if err := c.getWithOptions(ctx, path, &data, opts); err != nil {
		return Operation{}, err
	}
	return data, nil
}
