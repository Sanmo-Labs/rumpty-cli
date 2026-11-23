package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Find resolves a workspace by uid, slug or name from the user's workspaces.
// Exact uid and slug matches win before name matching, and a name that
// matches several workspaces is an error rather than a guess.
func Find(ctx context.Context, rt *app.Runtime, ref string) (api.Workspace, error) {
	workspaces, err := rt.API().ListWorkspaces(ctx)
	if err != nil {
		return api.Workspace{}, err
	}
	for i := range workspaces {
		if workspaces[i].UID == ref {
			return workspaces[i], nil
		}
	}
	for i := range workspaces {
		if workspaces[i].Slug == ref {
			return workspaces[i], nil
		}
	}
	var matches []api.Workspace
	for i := range workspaces {
		if strings.EqualFold(workspaces[i].Name, ref) {
			matches = append(matches, workspaces[i])
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Workspace{}, fmt.Errorf("workspace %q not found", ref)
	default:
		slugs := make([]string, 0, len(matches))
		for i := range matches {
			slugs = append(slugs, matches[i].Slug)
		}
		return api.Workspace{}, fmt.Errorf("workspace name %q is ambiguous (%s); use a uid or slug instead", ref, strings.Join(slugs, ", "))
	}
}

// resolveWorkspace returns the workspace to act on: the argument when set,
// otherwise the configured default.
func resolveWorkspace(ctx context.Context, rt *app.Runtime, ref string) (api.Workspace, error) {
	if strings.TrimSpace(ref) == "" {
		ref = rt.Config.Workspace
	}
	if strings.TrimSpace(ref) == "" {
		return api.Workspace{}, config.NewUsageError("no workspace given; pass one or set a default")
	}
	return Find(ctx, rt, ref)
}

// ListInvitations prints a workspace's pending invitations.
func ListInvitations(ctx context.Context, rt *app.Runtime, wsRef string, format term.OutputFormat) error {
	ws, err := resolveWorkspace(ctx, rt, wsRef)
	if err != nil {
		return err
	}

	spin := term.StartLoading(rt.Streams.ErrOut, "Loading invitations")
	invitations, err := rt.API().ListInvitations(ctx, ws.UID, ws.Slug)
	spin.Stop()
	if err != nil {
		return err
	}

	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, invitations)
	}
	if len(invitations) == 0 {
		fmt.Fprintln(w, "No pending invitations.")
		return nil
	}

	table := term.NewTable(w, "EMAIL", "ROLE", "STATUS", "EXPIRES")
	for i := range invitations {
		inv := &invitations[i]
		table.Row(inv.Email, inv.Role, inv.Status, inv.ExpiresAt)
	}
	if err := table.Render(); err != nil {
		return err
	}
	if len(invitations) == 1 {
		term.Summary(w, "1 pending invitation")
	} else {
		term.Summary(w, fmt.Sprintf("%d pending invitations", len(invitations)))
	}
	return nil
}

// RevokeInvitation revokes a pending invitation matched by uid or email.
func RevokeInvitation(ctx context.Context, rt *app.Runtime, wsRef, ref string) error {
	ws, err := resolveWorkspace(ctx, rt, wsRef)
	if err != nil {
		return err
	}

	spin := term.StartLoading(rt.Streams.ErrOut, "Finding invitation")
	invitations, err := rt.API().ListInvitations(ctx, ws.UID, ws.Slug)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findInvitation(invitations, ref, ws.Slug)
	if err != nil {
		return err
	}

	spin = term.StartLoading(rt.Streams.ErrOut, "Revoking invitation for "+target.Email)
	revoked, err := rt.API().RevokeInvitation(ctx, ws.UID, ws.Slug, target.UID)
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Revoked invitation for %s", revoked.Email)))
	return nil
}

func findInvitation(invitations []api.WorkspaceInvitation, ref, workspaceSlug string) (api.WorkspaceInvitation, error) {
	email := strings.ToLower(ref)
	var matches []api.WorkspaceInvitation
	for i := range invitations {
		inv := &invitations[i]
		if inv.UID == ref || inv.Email == email {
			matches = append(matches, *inv)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.WorkspaceInvitation{}, fmt.Errorf("no pending invitation %q in workspace %s", ref, workspaceSlug)
	default:
		uids := make([]string, 0, len(matches))
		for i := range matches {
			uids = append(uids, matches[i].UID)
		}
		return api.WorkspaceInvitation{}, fmt.Errorf("multiple invitations match %q: %s", ref, strings.Join(uids, ", "))
	}
}
