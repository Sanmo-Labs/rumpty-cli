package api

import (
	"context"
	"net/url"
)

// Cluster is a Kubernetes cluster in a workspace.
type Cluster struct {
	UID                  string `json:"uid"`
	Name                 string `json:"name"`
	Slug                 string `json:"slug"`
	Status               string `json:"status"`
	DisplayStatus        string `json:"display_status,omitempty"`
	ZoneSlug             string `json:"zone_slug,omitempty"`
	Version              string `json:"version,omitempty"`
	Endpoint             string `json:"endpoint,omitempty"`
	Kubeconfig           string `json:"kubeconfig,omitempty"`
	KubeconfigRestricted bool   `json:"kubeconfig_restricted,omitempty"`
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

// GetCluster returns one cluster, including its kubeconfig once available.
func (c *Client) GetCluster(ctx context.Context, workspace, uid string) (Cluster, error) {
	var data Cluster
	err := c.getWithOptions(ctx, "/v1/kubernetes/clusters/"+url.PathEscape(uid), &data, workspaceRequestOptions(workspace, ""))
	return data, err
}

// CreateClusterRequest is the body for creating a Kubernetes cluster. Every
// field except Name is optional.
type CreateClusterRequest struct {
	Name                 string `json:"name"`
	Version              string `json:"version,omitempty"`
	ImageSlug            string `json:"image_slug,omitempty"`
	ControlPlanePlanSlug string `json:"control_plane_plan_slug,omitempty"`
	WorkerPlanSlug       string `json:"worker_plan_slug,omitempty"`
	WorkerCount          int    `json:"worker_count,omitempty"`
}

// CreateCluster queues a new cluster and returns 202 with the create
// operation. Poll Get an operation or Get a cluster until it is running.
func (c *Client) CreateCluster(ctx context.Context, workspace string, req *CreateClusterRequest, idempotency string) (ClusterOperationResult, error) {
	var data ClusterOperationResult
	err := c.post(ctx, "/v1/kubernetes/clusters", req, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}
