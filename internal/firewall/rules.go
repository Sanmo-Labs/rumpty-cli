package firewall

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// RuleOpts carries the flags for adding a firewall rule.
type RuleOpts struct {
	Port        string
	PortEnd     string
	Source      string
	Direction   string
	Protocol    string
	Description string
}

// Add adds an allow rule to a firewall policy. The policy re-applies to
// every resource it is attached to.
func Add(ctx context.Context, rt *app.Runtime, policyRef string, opts *RuleOpts) error {
	if opts.Source == "" {
		return config.NewUsageError("--source is required (a CIDR block, such as 0.0.0.0/0)")
	}
	start, end, err := parsePorts(opts.Port, opts.PortEnd)
	if err != nil {
		return config.NewUsageError("%v", err)
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin := term.StartLoading(errOut(rt), "Finding firewall policy")
	policies, err := rt.API().ListFirewallPolicies(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	policy, err := findPolicy(policies, policyRef, workspace)
	if err != nil {
		return err
	}

	direction := strings.ToLower(opts.Direction)
	if direction == "" {
		direction = "inbound"
	}
	protocol := strings.ToLower(opts.Protocol)
	if protocol == "" {
		protocol = "tcp"
	}

	spin = term.StartLoading(errOut(rt), "Adding rule to "+policy.Name)
	result, err := rt.API().AddFirewallRule(ctx, workspace, policy.UID, &api.AddFirewallRuleRequest{
		Direction:   direction,
		Action:      "allow",
		Protocol:    protocol,
		PortStart:   start,
		PortEnd:     end,
		CIDR:        opts.Source,
		Description: opts.Description,
	})
	if err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	line := fmt.Sprintf("Added rule %s from %s to %s", describeRule(&result.FirewallRule), opts.Source, policy.Name)
	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, line))
	if result.OperationID != "" {
		spin = term.StartLoading(errOut(rt), "Applying the policy")
		err := waitForPolicyApply(ctx, rt, workspace, result.OperationID, spin)
		if err != nil {
			return err
		}
		spin.Stop()
		fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, "Applied to the attached resources"))
	}
	return nil
}

// RmRule deletes one rule from a firewall policy. The rule is matched by uid
// or a compact protocol/port reference such as tcp/22. Default rules can be
// deleted too; deleting the SSH rule from a policy attached to a VM can
// block access unless another policy still allows port 22.
func RmRule(ctx context.Context, rt *app.Runtime, policyRef, ruleRef string, yes bool) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Finding firewall policy")
	policies, err := rt.API().ListFirewallPolicies(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	policy, err := findPolicy(policies, policyRef, workspace)
	if err != nil {
		return err
	}
	rule, err := findRule(policy.Rules, ruleRef)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete rule %s from %s without confirmation; pass --yes to skip the prompt", describeRule(&rule), policy.Name)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete rule %s from %s?", describeRule(&rule), policy.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Deleting rule from "+policy.Name)
	if err := rt.API().DeleteFirewallRule(ctx, workspace, policy.UID, rule.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Deleted rule %s from %s", describeRule(&rule), policy.Name)))
	return nil
}

func findRule(rules []api.FirewallRule, ref string) (api.FirewallRule, error) {
	proto, start, end, isPattern, patternErr := parseRuleRef(ref)
	for i := range rules {
		if rules[i].UID == ref {
			return rules[i], nil
		}
		if isPattern && patternErr == nil && ruleMatches(&rules[i], proto, start, end) {
			return rules[i], nil
		}
	}
	if patternErr != nil {
		return api.FirewallRule{}, fmt.Errorf("invalid rule reference %q: %v", ref, patternErr)
	}
	descriptors := make([]string, 0, len(rules))
	for i := range rules {
		descriptors = append(descriptors, describeRule(&rules[i]))
	}
	return api.FirewallRule{}, fmt.Errorf("no rule %q; existing rules: %s", ref, strings.Join(descriptors, ", "))
}

// parseRuleRef reads a rule reference that is either a uid or a compact
// protocol/port form such as tcp/22, udp/53-63 or all/all.
func parseRuleRef(ref string) (string, int, int, bool, error) {
	idx := strings.Index(ref, "/")
	if idx < 0 {
		return "", 0, 0, false, nil
	}
	proto := strings.ToLower(ref[:idx])
	switch proto {
	case "tcp", "udp", "all":
	default:
		return "", 0, 0, true, fmt.Errorf("protocol must be tcp, udp, or all")
	}
	rest := ref[idx+1:]
	if rest == "all" {
		return proto, 0, 0, true, nil
	}
	start, end, err := parsePortRange(rest, "")
	return proto, start, end, true, err
}

func ruleMatches(rule *api.FirewallRule, proto string, start, end int) bool {
	if !strings.EqualFold(rule.Protocol, proto) {
		return false
	}
	if start == 0 && end == 0 {
		return rule.PortStart == 0 && rule.PortEnd == 0
	}
	return rule.PortStart == start && rule.PortEnd == end
}

func describeRule(rule *api.FirewallRule) string {
	proto := strings.ToLower(rule.Protocol)
	if rule.PortStart == 0 && rule.PortEnd == 0 {
		return proto + "/all"
	}
	if rule.PortStart == rule.PortEnd {
		return fmt.Sprintf("%s/%d", proto, rule.PortStart)
	}
	return fmt.Sprintf("%s/%d-%d", proto, rule.PortStart, rule.PortEnd)
}

// parsePorts reads the --port flag: empty for all ports, one number, or a
// start-end range.
func parsePorts(port, portEnd string) (int, int, error) {
	if strings.TrimSpace(port) == "" {
		if strings.TrimSpace(portEnd) != "" {
			return 0, 0, fmt.Errorf("--port-end given without --port")
		}
		return 0, 0, nil
	}
	return parsePortRange(port, portEnd)
}

func parsePortRange(spec, portEnd string) (int, int, error) {
	if strings.Contains(spec, "-") {
		parts := strings.SplitN(spec, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("port %q is not a number", parts[0])
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("port %q is not a number", parts[1])
		}
		return start, end, nil
	}
	start, err := strconv.Atoi(strings.TrimSpace(spec))
	if err != nil {
		return 0, 0, fmt.Errorf("port %q is not a number", spec)
	}
	if strings.TrimSpace(portEnd) != "" {
		end, err := strconv.Atoi(strings.TrimSpace(portEnd))
		if err != nil {
			return 0, 0, fmt.Errorf("port %q is not a number", portEnd)
		}
		return start, end, nil
	}
	return start, start, nil
}
