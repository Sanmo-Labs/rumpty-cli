package workspace

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

var memberRoles = map[string]bool{"admin": true, "developer": true, "viewer": true}

// Invite emails an invitation to join the workspace. The token in the email
// never comes back through the API.
func Invite(ctx context.Context, rt *app.Runtime, wsRef, email, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if !memberRoles[role] {
		return fmt.Errorf("role must be admin, developer or viewer, got %q", role)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	w := errOut(rt)
	ws, err := Find(ctx, rt, wsRef)
	if err != nil {
		return err
	}

	spin := term.StartLoading(w, "Inviting "+email)
	invitation, err := rt.API().InviteWorkspaceMember(ctx, ws.UID, ws.Slug, api.InviteWorkspaceMemberRequest{
		Email: email,
		Role:  role,
	})
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Invited %s as %s (expires %s)", invitation.Email, invitation.Role, invitation.ExpiresAt)))
	return nil
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
