package firewall

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

func TestFindRule_ambiguousPatternRequiresUID(t *testing.T) {
	rules := []api.FirewallRule{
		{UID: "rule-1", Protocol: "tcp", PortStart: 22, PortEnd: 22, CIDR: "10.0.0.0/8"},
		{UID: "rule-2", Protocol: "tcp", PortStart: 22, PortEnd: 22, CIDR: "0.0.0.0/0"},
		{UID: "rule-3", Protocol: "udp", PortStart: 53, PortEnd: 53},
	}

	_, err := findRule(rules, "tcp/22")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches 2 rules")
	assert.Contains(t, err.Error(), "rule-1")
	assert.Contains(t, err.Error(), "rule-2")
}

func TestFindRule_uniquePatternMatches(t *testing.T) {
	rules := []api.FirewallRule{
		{UID: "rule-1", Protocol: "tcp", PortStart: 22, PortEnd: 22},
		{UID: "rule-2", Protocol: "udp", PortStart: 53, PortEnd: 53},
	}

	rule, err := findRule(rules, "udp/53")
	require.NoError(t, err)
	assert.Equal(t, "rule-2", rule.UID)
}

func TestFindRule_uidWins(t *testing.T) {
	rules := []api.FirewallRule{
		{UID: "tcp/22", Protocol: "tcp", PortStart: 80, PortEnd: 80},
		{UID: "rule-2", Protocol: "tcp", PortStart: 22, PortEnd: 22},
	}

	rule, err := findRule(rules, "tcp/22")
	require.NoError(t, err)
	assert.Equal(t, "tcp/22", rule.UID)
}

func TestFindRule_noMatchListsRules(t *testing.T) {
	rules := []api.FirewallRule{
		{UID: "rule-1", Protocol: "tcp", PortStart: 22, PortEnd: 22},
	}

	_, err := findRule(rules, "udp/53")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no rule")
	assert.Contains(t, err.Error(), "tcp/22")
}

func TestFindRule_invalidPattern(t *testing.T) {
	_, err := findRule(nil, "icmp/22")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rule reference")
}
