package api

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

const headerWorkspaceSlug = "X-Workspace-Slug"
const headerIdempotency = "X-Idempotency"

func workspaceRequestOptions(workspace, idempotency string) requestOptions {
	return requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
			headerIdempotency:   idempotency,
		},
	}
}

func (c *Client) ListVMs(ctx context.Context, workspace string) ([]VM, error) {
	var data []VM
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
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

func (c *Client) ListVMApps(ctx context.Context, workspace, vmUID string) ([]VMApp, error) {
	var data []VMApp
	if err := c.getWithOptions(ctx, vmPath(vmUID, "apps"), &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}); err != nil {
		return nil, err
	}
	if data == nil {
		return []VMApp{}, nil
	}
	return data, nil
}

func vmPath(vmUID, action string) string {
	path := "/v1/vms/" + url.PathEscape(vmUID)
	if action != "" {
		path += "/" + action
	}
	return path
}

func (c *Client) StartVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error) {
	var data VMOperationResult
	err := c.post(ctx, vmPath(vmUID, "start"), nil, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) StopVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error) {
	var data VMOperationResult
	err := c.post(ctx, vmPath(vmUID, "stop"), nil, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) RebootVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error) {
	var data VMOperationResult
	err := c.post(ctx, vmPath(vmUID, "reboot"), nil, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) DeleteVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error) {
	var data VMOperationResult
	err := c.deleteWithOptions(ctx, vmPath(vmUID, ""), &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) ExposeVMApp(ctx context.Context, workspace, vmUID string, req ExposeVMAppRequest, idempotency string) (ExposeVMAppResult, error) {
	var data ExposeVMAppResult
	err := c.post(ctx, vmPath(vmUID, "apps"), req, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) UnexposeVMApp(ctx context.Context, workspace, vmUID, app string, idempotency string) (UnexposeVMAppResult, error) {
	var data UnexposeVMAppResult
	path := vmPath(vmUID, "apps") + "/" + url.PathEscape(app)
	err := c.deleteWithOptions(ctx, path, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

// NewIdempotencyKey returns a unique key for a single VM lifecycle request.
func NewIdempotencyKey() string {
	return fmt.Sprintf("rumpty-cli-%d", time.Now().UnixNano())
}
