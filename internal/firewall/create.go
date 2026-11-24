package firewall

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

// Create makes a new firewall policy with its two default rules.
func Create(ctx context.Context, rt *app.Runtime, name, description string) error {
	w := errOut(rt)
	spin := term.StartLoading(w, "Creating firewall policy "+name)
	policy, err := rt.API().CreateFirewallPolicy(ctx, strings.TrimSpace(rt.Config.Workspace), api.CreateFirewallPolicyRequest{
		Name:        name,
		Description: description,
	})
	spin.Stop()
	if err != nil {
		return err
	}

	line := fmt.Sprintf("Created firewall policy %s", policy.Name)
	if description != "" {
		line = fmt.Sprintf("%s (%s)", line, description)
	}
	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, line))
	return nil
}

// Rm deletes a firewall policy matched by uid, name or slug.
func Rm(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding firewall policy")
	policies, err := rt.API().ListFirewallPolicies(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findPolicy(policies, ref, workspace)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete firewall policy %s without confirmation; pass --yes to skip the prompt", target.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete firewall policy %s? Attached resources lose it and their firewalls are rebuilt", target.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Deleting "+target.Name)
	if err := rt.API().DeleteFirewallPolicy(ctx, workspace, target.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Deleted firewall policy %s", target.Name)))
	return nil
}

func findPolicy(policies []api.FirewallPolicy, ref, workspaceSlug string) (api.FirewallPolicy, error) {
	var matches []api.FirewallPolicy
	for i := range policies {
		p := &policies[i]
		if p.UID == ref || p.Name == ref || p.Slug == ref {
			matches = append(matches, *p)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.FirewallPolicy{}, fmt.Errorf("no firewall policy %q in workspace %s", ref, workspaceSlug)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Name)
		}
		return api.FirewallPolicy{}, fmt.Errorf("multiple firewall policies match %q: %s", ref, strings.Join(names, ", "))
	}
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
