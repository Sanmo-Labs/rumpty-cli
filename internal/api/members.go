package api

import (
	"context"
	"fmt"
	"net/url"
)

// WorkspaceMember is a member of a workspace.
type WorkspaceMember struct {
	UID           string `json:"uid"`
	UserUID       string `json:"user_uid"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	JoinedAt      string `json:"joined_at,omitempty"`
	IsCurrentUser bool   `json:"is_current_user,omitempty"`
}

// InviteWorkspaceMemberRequest is the body for inviting a member.
type InviteWorkspaceMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ListWorkspaceMembers returns a workspace's members.
func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceUID, workspaceSlug string) ([]WorkspaceMember, error) {
	var data []WorkspaceMember
	if err := c.getWithOptions(ctx, workspaceMembersPath(workspaceUID), &data, workspaceRequestOptions(workspaceSlug, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []WorkspaceMember{}, nil
	}
	return data, nil
}

// InviteWorkspaceMember emails an invitation to join the workspace. The
// token in the email never comes back through the API.
func (c *Client) InviteWorkspaceMember(ctx context.Context, workspaceUID, workspaceSlug string, req InviteWorkspaceMemberRequest) (WorkspaceInvitation, error) {
	var data WorkspaceInvitation
	err := c.post(ctx, workspaceInvitationsPath(workspaceUID), req, &data, workspaceRequestOptions(workspaceSlug, ""))
	return data, err
}

// RemoveWorkspaceMember removes a member immediately. The response is the
// member as they were before removal.
func (c *Client) RemoveWorkspaceMember(ctx context.Context, workspaceUID, workspaceSlug, memberUID string) (WorkspaceMember, error) {
	var data WorkspaceMember
	err := c.deleteWithOptions(ctx, workspaceMemberPath(workspaceUID, memberUID), &data, workspaceRequestOptions(workspaceSlug, ""))
	return data, err
}

// UpdateWorkspaceMemberRoleRequest is the body for changing a member's role.
type UpdateWorkspaceMemberRoleRequest struct {
	Role string `json:"role"`
}

// UpdateWorkspaceMemberRole changes a member's role. The new permissions
// apply to the member's next request.
func (c *Client) UpdateWorkspaceMemberRole(ctx context.Context, workspaceUID, workspaceSlug, memberUID, role string) (WorkspaceMember, error) {
	var data WorkspaceMember
	err := c.patchWithOptions(ctx, workspaceMemberPath(workspaceUID, memberUID),
		UpdateWorkspaceMemberRoleRequest{Role: role}, &data, workspaceRequestOptions(workspaceSlug, ""))
	return data, err
}

func workspaceMembersPath(workspaceUID string) string {
	return fmt.Sprintf("/v1/workspaces/%s/members", url.PathEscape(workspaceUID))
}

func workspaceMemberPath(workspaceUID, memberUID string) string {
	return fmt.Sprintf("%s/%s", workspaceMembersPath(workspaceUID), url.PathEscape(memberUID))
}

func (c *Client) patchWithOptions(ctx context.Context, path string, body any, out any, opts requestOptions) error {
	if c.err != nil {
		return c.err
	}
	req := c.r.R().SetContext(ctx)
	req.SetBody(body)
	for k, v := range opts.headers {
		req.SetHeader(k, v)
	}
	resp, err := req.Patch(path)
	if err != nil {
		return TransportError(err)
	}
	return decodeEnvelope(resp, out)
}
