package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// RmMember removes a member matched by username, email or membership uid.
func RmMember(ctx context.Context, rt *app.Runtime, wsRef, ref string, yes bool) error {
	w := errOut(rt)
	ws, err := Find(ctx, rt, wsRef)
	if err != nil {
		return err
	}
	spin := term.StartLoading(w, "Finding member")
	members, err := rt.API().ListWorkspaceMembers(ctx, ws.UID, ws.Slug)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findMember(members, ref)
	if err != nil {
		return err
	}
	if target.IsCurrentUser {
		return fmt.Errorf("cannot remove yourself; ask the workspace owner to remove you")
	}
	if target.Role == "owner" {
		return fmt.Errorf("the workspace owner cannot be removed")
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to remove %s without confirmation; pass --yes to skip the prompt", target.Username)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Remove %s from %s? They lose access immediately", target.Username, ws.Slug))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Removing "+target.Username)
	_, err = rt.API().RemoveWorkspaceMember(ctx, ws.UID, ws.Slug, target.UID)
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Removed %s", target.Username)))
	return nil
}

// SetRole changes a member's role to admin, developer or viewer.
func SetRole(ctx context.Context, rt *app.Runtime, wsRef, ref, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if !memberRoles[role] {
		return fmt.Errorf("role must be admin, developer or viewer, got %q", role)
	}

	w := errOut(rt)
	ws, err := Find(ctx, rt, wsRef)
	if err != nil {
		return err
	}
	spin := term.StartLoading(w, "Finding member")
	members, err := rt.API().ListWorkspaceMembers(ctx, ws.UID, ws.Slug)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findMember(members, ref)
	if err != nil {
		return err
	}
	if target.Role == "owner" {
		return fmt.Errorf("the owner's role cannot be changed")
	}

	spin = term.StartLoading(w, "Changing "+target.Username+"'s role")
	_, err = rt.API().UpdateWorkspaceMemberRole(ctx, ws.UID, ws.Slug, target.UID, role)
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("%s is now %s", target.Username, role)))
	return nil
}

func findMember(members []api.WorkspaceMember, ref string) (api.WorkspaceMember, error) {
	var matches []api.WorkspaceMember
	for i := range members {
		m := &members[i]
		if m.UID == ref || m.UserUID == ref || strings.EqualFold(m.Username, ref) || strings.EqualFold(m.Email, ref) {
			matches = append(matches, *m)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.WorkspaceMember{}, fmt.Errorf("no member %q in the workspace", ref)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Username)
		}
		return api.WorkspaceMember{}, fmt.Errorf("multiple members match %q: %s", ref, strings.Join(names, ", "))
	}
}
