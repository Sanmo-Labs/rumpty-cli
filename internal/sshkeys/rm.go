package sshkeys

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Rm removes an SSH public key matched by uid, name or fingerprint.
func Rm(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding ssh key")
	keys, err := rt.API().ListSSHKeys(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findSSHKey(keys, ref, workspace)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to remove ssh key %s without confirmation; pass --yes to skip the prompt", target.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Remove ssh key %s (%s)?", target.Name, target.Fingerprint))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Removing "+target.Name)
	if err := rt.API().DeleteSSHKey(ctx, workspace, target.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Removed %s (%s)", target.Name, target.Fingerprint)))
	return nil
}

func findSSHKey(keys []api.SSHKey, ref, workspaceSlug string) (api.SSHKey, error) {
	fingerprint := strings.TrimPrefix(ref, "SHA256:")
	var matches []api.SSHKey
	for i := range keys {
		key := &keys[i]
		if key.UID == ref || key.Name == ref || strings.TrimPrefix(key.Fingerprint, "SHA256:") == fingerprint {
			matches = append(matches, *key)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.SSHKey{}, fmt.Errorf("no ssh key %q in workspace %s", ref, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.SSHKey{}, fmt.Errorf("multiple ssh keys match %q: %s", ref, strings.Join(names, ", "))
	}
}
