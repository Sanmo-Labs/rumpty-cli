package api

import (
	"context"
	"net/url"
)

type LoadBalancerHealthCheckInput struct {
	Protocol           *string `json:"protocol,omitempty"`
	Path               *string `json:"path,omitempty"`
	IntervalSeconds    *int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds     *int    `json:"timeout_seconds,omitempty"`
	HealthyThreshold   *int    `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold *int    `json:"unhealthy_threshold,omitempty"`
}

func (h *LoadBalancerHealthCheckInput) IsZero() bool {
	return h == nil || *h == LoadBalancerHealthCheckInput{}
}

type CreateLoadBalancerRequest struct {
	Name           string                        `json:"name"`
	TargetPort     int                           `json:"target_port"`
	TargetProtocol string                        `json:"target_protocol,omitempty"`
	VMUIDs         []string                      `json:"vm_uids"`
	HealthCheck    *LoadBalancerHealthCheckInput `json:"health_check,omitempty"`
	StickySessions *bool                         `json:"sticky_sessions,omitempty"`
	AllowedCIDRs   []string                      `json:"allowed_cidrs,omitempty"`
}

type UpdateLoadBalancerRequest struct {
	TargetPort     *int                          `json:"target_port,omitempty"`
	TargetProtocol *string                       `json:"target_protocol,omitempty"`
	VMUIDs         *[]string                     `json:"vm_uids,omitempty"`
	HealthCheck    *LoadBalancerHealthCheckInput `json:"health_check,omitempty"`
	StickySessions *bool                         `json:"sticky_sessions,omitempty"`
	AllowedCIDRs   *[]string                     `json:"allowed_cidrs,omitempty"`
}

type LoadBalancerTarget struct {
	VMUID    string `json:"vm_uid"`
	VMName   string `json:"vm_name"`
	VMStatus string `json:"vm_status"`
}

type LoadBalancerHealthCheck struct {
	Protocol           string `json:"protocol"`
	Path               string `json:"path,omitempty"`
	Port               int    `json:"port"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	HealthyThreshold   int    `json:"healthy_threshold"`
	UnhealthyThreshold int    `json:"unhealthy_threshold"`
}

type LoadBalancerStickySessions struct {
	Enabled    bool   `json:"enabled"`
	CookieName string `json:"cookie_name"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type LoadBalancer struct {
	UID            string                     `json:"uid"`
	Name           string                     `json:"name"`
	Slug           string                     `json:"slug"`
	Status         string                     `json:"status"`
	Hostname       string                     `json:"hostname"`
	URL            string                     `json:"url"`
	TargetPort     int                        `json:"target_port"`
	TargetProtocol string                     `json:"target_protocol"`
	Targets        []LoadBalancerTarget       `json:"targets"`
	HealthCheck    LoadBalancerHealthCheck    `json:"health_check"`
	StickySessions LoadBalancerStickySessions `json:"sticky_sessions"`
	AllowedCIDRs   []string                   `json:"allowed_cidrs"`
	OperationID    string                     `json:"operation_id,omitempty"`
	LastFailure    string                     `json:"last_failure,omitempty"`
	CreatedAt      string                     `json:"created_at"`
	UpdatedAt      string                     `json:"updated_at"`
}

type LoadBalancerOperationResult struct {
	OperationID  string        `json:"operation_id"`
	Status       string        `json:"status"`
	EventsURL    string        `json:"events_url"`
	IsReplay     bool          `json:"is_replay"`
	IsCompleted  bool          `json:"is_completed"`
	LoadBalancer *LoadBalancer `json:"load_balancer,omitempty"`
}

type loadBalancerTargetsRequest struct {
	VMUIDs []string `json:"vm_uids"`
}

func workspaceReadOptions(workspace string) requestOptions {
	return requestOptions{headers: map[string]string{headerWorkspaceSlug: workspace}}
}

func loadBalancerPath(uid, action string) string {
	path := "/v1/load-balancers"
	if uid != "" {
		path += "/" + url.PathEscape(uid)
	}
	if action != "" {
		path += "/" + action
	}
	return path
}

func (c *Client) ListLoadBalancers(ctx context.Context, workspace string) ([]LoadBalancer, error) {
	var data []LoadBalancer
	if err := c.getWithOptions(ctx, loadBalancerPath("", ""), &data, workspaceReadOptions(workspace)); err != nil {
		return nil, err
	}
	if data == nil {
		return []LoadBalancer{}, nil
	}
	return data, nil
}

func (c *Client) GetLoadBalancer(ctx context.Context, workspace, uid string) (LoadBalancer, error) {
	var data LoadBalancer
	err := c.getWithOptions(ctx, loadBalancerPath(uid, ""), &data, workspaceReadOptions(workspace))
	return data, err
}

func (c *Client) CreateLoadBalancer(ctx context.Context, workspace string, req *CreateLoadBalancerRequest, idempotency string) (LoadBalancerOperationResult, error) {
	var data LoadBalancerOperationResult
	err := c.post(ctx, loadBalancerPath("", ""), req, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) UpdateLoadBalancer(ctx context.Context, workspace, uid string, req *UpdateLoadBalancerRequest, idempotency string) (LoadBalancerOperationResult, error) {
	var data LoadBalancerOperationResult
	err := c.patch(ctx, loadBalancerPath(uid, ""), req, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) DeleteLoadBalancer(ctx context.Context, workspace, uid, idempotency string) (LoadBalancerOperationResult, error) {
	var data LoadBalancerOperationResult
	err := c.deleteWithOptions(ctx, loadBalancerPath(uid, ""), &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) AddLoadBalancerTargets(ctx context.Context, workspace, uid string, vmUIDs []string, idempotency string) (LoadBalancerOperationResult, error) {
	var data LoadBalancerOperationResult
	err := c.post(ctx, loadBalancerPath(uid, "targets"), loadBalancerTargetsRequest{VMUIDs: vmUIDs}, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}

func (c *Client) RemoveLoadBalancerTarget(ctx context.Context, workspace, uid, vmUID, idempotency string) (LoadBalancerOperationResult, error) {
	var data LoadBalancerOperationResult
	path := loadBalancerPath(uid, "targets") + "/" + url.PathEscape(vmUID)
	err := c.deleteWithOptions(ctx, path, &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}
