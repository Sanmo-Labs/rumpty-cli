package api

import (
	"context"
	"net/url"
)

// Cluster is a Kubernetes cluster in a workspace.
type Cluster struct {
	UID           string `json:"uid"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Status        string `json:"status"`
	DisplayStatus string `json:"display_status,omitempty"`
	ZoneSlug      string `json:"zone_slug,omitempty"`
	Version       string `json:"version,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`
}

// ClusterOperationResult is returned when a cluster operation is queued.
type ClusterOperationResult struct {
	OperationID string  `json:"operation_id"`
	Cluster     Cluster `json:"cluster"`
}

// ListClusters returns the workspace's clusters.
func (c *Client) ListClusters(ctx context.Context, workspace string) ([]Cluster, error) {
	var data []Cluster
	if err := c.getWithOptions(ctx, "/v1/kubernetes/clusters", &data, workspaceRequestOptions(workspace, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []Cluster{}, nil
	}
	return data, nil
}

// DeleteCluster queues deletion of the cluster and all of its nodes. It
// returns the operation to wait on.
func (c *Client) DeleteCluster(ctx context.Context, workspace, uid string) (ClusterOperationResult, error) {
	var data ClusterOperationResult
	err := c.deleteWithOptions(ctx, "/v1/kubernetes/clusters/"+url.PathEscape(uid), &data, workspaceRequestOptions(workspace, NewIdempotencyKey()))
	return data, err
}
