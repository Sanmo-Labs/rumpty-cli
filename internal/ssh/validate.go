package ssh

import (
	"fmt"
	"regexp"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

var (
	sessionHostPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.:-]*$`)
	sessionNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._+-]*$`)
)

// validateSession rejects certificate responses whose values could be read
// as shell syntax or ssh options when building the ssh command line.
func validateSession(s *api.CertResponse) error {
	fields := []struct {
		name, value string
		pattern     *regexp.Regexp
	}{
		{"edge_host", s.EdgeHost, sessionHostPattern},
		{"router_user", s.RouterUser, sessionNamePattern},
		{"username", s.Username, sessionNamePattern},
		{"vm_slug", s.VMSlug, sessionNamePattern},
	}
	for _, f := range fields {
		if !f.pattern.MatchString(f.value) {
			return fmt.Errorf("ssh certificate response has an invalid %s %q", f.name, f.value)
		}
	}
	if s.EdgePort < 1 || s.EdgePort > 65535 {
		return fmt.Errorf("ssh certificate response has an invalid edge_port %d", s.EdgePort)
	}
	return nil
}
