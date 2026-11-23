package api

import (
	"context"
	"fmt"
	"net/url"
)

// Volume is a block-storage volume in a workspace.
type Volume struct {
	UID        string `json:"uid"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	ZoneSlug   string `json:"zone_slug"`
	SizeGiB    int    `json:"size_gib"`
	Filesystem string `json:"filesystem"`
	Formatted  bool   `json:"formatted"`
	VmUID      string `json:"vm_uid,omitempty"`
	VmName     string `json:"vm_name,omitempty"`
}

// CreateVolumeRequest is the body for creating a volume.
type CreateVolumeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SizeGiB     int    `json:"size_gib"`
	ZoneSlug    string `json:"zone_slug,omitempty"`
	Filesystem  string `json:"filesystem,omitempty"`
}

// VolumeOperationResult is returned when a volume operation is queued.
type VolumeOperationResult struct {
	OperationID string `json:"operation_id"`
	Volume      Volume `json:"volume"`
}

// ListVolumes returns the workspace's volumes.
func (c *Client) ListVolumes(ctx context.Context, workspace string) ([]Volume, error) {
	var data []Volume
	if err := c.getWithOptions(ctx, "/v1/volumes", &data, workspaceRequestOptions(workspace, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []Volume{}, nil
	}
	return data, nil
}

// CreateVolume queues creation of a volume. It returns the operation to wait
// on and the volume in its creating state.
func (c *Client) CreateVolume(ctx context.Context, workspace string, req CreateVolumeRequest) (VolumeOperationResult, error) {
	var data VolumeOperationResult
	err := c.post(ctx, "/v1/volumes", req, &data, workspaceRequestOptions(workspace, NewIdempotencyKey()))
	return data, err
}

// DeleteVolume queues deletion of a volume. Everything stored on it is
// destroyed; detach it first.
func (c *Client) DeleteVolume(ctx context.Context, workspace, uid string) (VolumeOperationResult, error) {
	var data VolumeOperationResult
	err := c.deleteWithOptions(ctx, "/v1/volumes/"+url.PathEscape(uid), &data, workspaceRequestOptions(workspace, NewIdempotencyKey()))
	return data, err
}

// AttachVolume queues attaching an available volume to a VM. Both must be
// in the same zone.
func (c *Client) AttachVolume(ctx context.Context, workspace, volumeUID, vmUID string) (VolumeOperationResult, error) {
	var data VolumeOperationResult
	err := c.post(ctx, fmt.Sprintf("/v1/volumes/%s/attach", url.PathEscape(volumeUID)),
		map[string]string{"vm_uid": vmUID}, &data, workspaceRequestOptions(workspace, NewIdempotencyKey()))
	return data, err
}

// DetachVolume queues detaching an in-use volume from its VM. Its data is
// kept; unmount inside the VM first to avoid losing unwritten data.
func (c *Client) DetachVolume(ctx context.Context, workspace, volumeUID string) (VolumeOperationResult, error) {
	var data VolumeOperationResult
	err := c.post(ctx, fmt.Sprintf("/v1/volumes/%s/detach", url.PathEscape(volumeUID)),
		nil, &data, workspaceRequestOptions(workspace, NewIdempotencyKey()))
	return data, err
}
