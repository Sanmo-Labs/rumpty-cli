package api

import (
	"context"
	"fmt"
	"net/url"
)

// FirewallRule is a rule inside a firewall policy.
type FirewallRule struct {
	UID         string `json:"uid"`
	Direction   string `json:"direction"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	PortStart   int    `json:"port_start"`
	PortEnd     int    `json:"port_end"`
	CIDR        string `json:"cidr"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

// FirewallPolicy is a firewall policy in a workspace.
type FirewallPolicy struct {
	UID         string         `json:"uid"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description,omitempty"`
	IsDefault   bool           `json:"is_default"`
	Rules       []FirewallRule `json:"rules,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
}

// CreateFirewallPolicyRequest is the body for creating a firewall policy.
type CreateFirewallPolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// ListFirewallPolicies returns the workspace's firewall policies.
func (c *Client) ListFirewallPolicies(ctx context.Context, workspace string) ([]FirewallPolicy, error) {
	var data []FirewallPolicy
	if err := c.getWithOptions(ctx, "/v1/firewall-policies", &data, workspaceRequestOptions(workspace, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []FirewallPolicy{}, nil
	}
	return data, nil
}

// CreateFirewallPolicy creates a policy with its two default rules.
func (c *Client) CreateFirewallPolicy(ctx context.Context, workspace string, req CreateFirewallPolicyRequest) (FirewallPolicy, error) {
	var data FirewallPolicy
	err := c.post(ctx, "/v1/firewall-policies", req, &data, workspaceRequestOptions(workspace, ""))
	return data, err
}

// DeleteFirewallPolicy deletes a policy and all of its attachments in one
// step.
func (c *Client) DeleteFirewallPolicy(ctx context.Context, workspace, uid string) error {
	return c.deleteWithOptions(ctx, "/v1/firewall-policies/"+url.PathEscape(uid), nil, workspaceRequestOptions(workspace, ""))
}

// AddFirewallRuleRequest is the body for adding a rule to a firewall policy.
type AddFirewallRuleRequest struct {
	Direction   string `json:"direction,omitempty"`
	Action      string `json:"action,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	PortStart   int    `json:"port_start,omitempty"`
	PortEnd     int    `json:"port_end,omitempty"`
	CIDR        string `json:"cidr"`
	Description string `json:"description,omitempty"`
}

// FirewallRuleResult is the response for adding a firewall rule. OperationID
// is the first firewall.apply operation queued for the change, or empty when
// the policy is not attached to anything.
type FirewallRuleResult struct {
	FirewallRule
	OperationID string `json:"operation_id"`
}

// AddFirewallRule adds an allow rule to a firewall policy.
func (c *Client) AddFirewallRule(ctx context.Context, workspace, policyUID string, req *AddFirewallRuleRequest) (FirewallRuleResult, error) {
	var data FirewallRuleResult
	err := c.post(ctx, fmt.Sprintf("/v1/firewall-policies/%s/rules", url.PathEscape(policyUID)),
		req, &data, workspaceRequestOptions(workspace, ""))
	return data, err
}

// DeleteFirewallRule deletes a rule from a firewall policy, including the
// default SSH and app-port rules.
func (c *Client) DeleteFirewallRule(ctx context.Context, workspace, policyUID, ruleUID string) error {
	return c.deleteWithOptions(ctx, fmt.Sprintf("/v1/firewall-policies/%s/rules/%s",
		url.PathEscape(policyUID), url.PathEscape(ruleUID)), nil, workspaceRequestOptions(workspace, ""))
}
