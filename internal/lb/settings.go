package lb

import (
	"fmt"
	"net"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

// MaxTargets is the most VMs one load balancer can send traffic to.
const MaxTargets = 10

// Settings holds the load balancer fields a command sets. Nil fields are left
// to the server default on create and unchanged on update.
type Settings struct {
	VMs                []string
	Port               *int
	Protocol           *string
	HealthProtocol     *string
	HealthPath         *string
	HealthInterval     *int
	HealthTimeout      *int
	HealthyThreshold   *int
	UnhealthyThreshold *int
	Sticky             *bool
	AllowedCIDRs       *[]string
}

func (s *Settings) Validate() error {
	if s.Port != nil && (*s.Port < 1 || *s.Port > 65535) {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if s.Protocol != nil && *s.Protocol != "http" && *s.Protocol != "grpc" {
		return fmt.Errorf("protocol must be http or grpc")
	}
	if s.HealthProtocol != nil && *s.HealthProtocol != "http" && *s.HealthProtocol != "tcp" {
		return fmt.Errorf("health check protocol must be http or tcp")
	}
	if s.Protocol != nil && *s.Protocol == "grpc" && s.HealthProtocol != nil && *s.HealthProtocol == "http" {
		return fmt.Errorf("grpc load balancers use tcp health checks; drop --health-protocol or set it to tcp")
	}
	if s.HealthPath != nil && ((s.HealthProtocol != nil && *s.HealthProtocol == "tcp") || (s.Protocol != nil && *s.Protocol == "grpc")) {
		return fmt.Errorf("--health-path only applies to http health checks; tcp and grpc checks only open a connection")
	}
	if s.HealthPath != nil && !strings.HasPrefix(*s.HealthPath, "/") {
		return fmt.Errorf("health check path must start with /")
	}
	if s.HealthInterval != nil && (*s.HealthInterval < 1 || *s.HealthInterval > 300) {
		return fmt.Errorf("health check interval must be between 1 and 300 seconds")
	}
	if s.HealthTimeout != nil && *s.HealthTimeout < 1 {
		return fmt.Errorf("health check timeout must be at least 1 second")
	}
	if s.HealthInterval != nil && s.HealthTimeout != nil && *s.HealthTimeout > *s.HealthInterval {
		return fmt.Errorf("health check timeout must not exceed the interval")
	}
	for _, t := range []*int{s.HealthyThreshold, s.UnhealthyThreshold} {
		if t != nil && (*t < 1 || *t > 10) {
			return fmt.Errorf("health check thresholds must be between 1 and 10")
		}
	}
	if s.AllowedCIDRs != nil {
		for _, cidr := range *s.AllowedCIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return fmt.Errorf("invalid CIDR %q", cidr)
			}
		}
	}
	return nil
}

func (s *Settings) healthCheck() *api.LoadBalancerHealthCheckInput {
	hc := &api.LoadBalancerHealthCheckInput{
		Protocol:           s.HealthProtocol,
		Path:               s.HealthPath,
		IntervalSeconds:    s.HealthInterval,
		TimeoutSeconds:     s.HealthTimeout,
		HealthyThreshold:   s.HealthyThreshold,
		UnhealthyThreshold: s.UnhealthyThreshold,
	}
	if hc.IsZero() {
		return nil
	}
	return hc
}

func (s *Settings) allowedCIDRs() *[]string {
	if s.AllowedCIDRs == nil {
		return nil
	}
	cidrs := append([]string{}, *s.AllowedCIDRs...)
	return &cidrs
}

func (s *Settings) IsEmpty() bool {
	return s.VMs == nil && s.Port == nil && s.Protocol == nil && s.healthCheck() == nil &&
		s.Sticky == nil && s.AllowedCIDRs == nil
}
