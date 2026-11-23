package api

import (
	"context"
	"fmt"
	"net/url"
)

// WorkspaceInvitation is a pending invitation to join a workspace.
type WorkspaceInvitation struct {
	UID       string     `json:"uid"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at,omitempty"`
	ExpiresAt string     `json:"expires_at,omitempty"`
	InvitedBy *InvitedBy `json:"invited_by,omitempty"`
}

// InvitedBy is the member who sent an invitation.
type InvitedBy struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
}

// ListInvitations returns a workspace's pending invitations.
func (c *Client) ListInvitations(ctx context.Context, workspaceUID, workspaceSlug string) ([]WorkspaceInvitation, error) {
	var data []WorkspaceInvitation
	if err := c.getWithOptions(ctx, workspaceInvitationsPath(workspaceUID), &data, workspaceRequestOptions(workspaceSlug, "")); err != nil {
		return nil, err
	}
	if data == nil {
		return []WorkspaceInvitation{}, nil
	}
	return data, nil
}

// RevokeInvitation revokes a pending invitation. Its link stops working
// straight away.
func (c *Client) RevokeInvitation(ctx context.Context, workspaceUID, workspaceSlug, invitationUID string) (WorkspaceInvitation, error) {
	var data WorkspaceInvitation
	err := c.deleteWithOptions(ctx,
		fmt.Sprintf("%s/%s", workspaceInvitationsPath(workspaceUID), url.PathEscape(invitationUID)),
		&data, workspaceRequestOptions(workspaceSlug, ""))
	return data, err
}

func workspaceInvitationsPath(workspaceUID string) string {
	return fmt.Sprintf("/v1/workspaces/%s/invitations", url.PathEscape(workspaceUID))
}
