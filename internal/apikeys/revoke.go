package apikeys

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

// Revoke revokes an API key matched by uid, name or token prefix.
func Revoke(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	spin := term.StartLoading(w, "Finding API key")
	keys, err := rt.API().ListAPIKeys(ctx)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findAPIKey(keys, ref)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to revoke API key %s without confirmation; pass --yes to skip the prompt", target.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Revoke API key %s (prefix %s)? Anything still using it stops working", target.Name, target.Prefix))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Revoking "+target.Name)
	if err := rt.API().RevokeAPIKey(ctx, target.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Revoked API key %s", target.Name)))
	return nil
}

func findAPIKey(keys []api.APIKey, ref string) (api.APIKey, error) {
	var matches []api.APIKey
	for i := range keys {
		key := &keys[i]
		if key.UID == ref || key.Name == ref || key.Prefix == ref {
			matches = append(matches, *key)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.APIKey{}, fmt.Errorf("no API key %q; names are matched exactly", ref)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.APIKey{}, fmt.Errorf("multiple API keys match %q: %s", ref, strings.Join(names, ", "))
	}
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
