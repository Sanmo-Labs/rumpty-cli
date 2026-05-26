package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
)

type Error struct {
	StatusCode int
	Message    string
	Action     string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed HTTP %d", e.StatusCode)
}

func UserMessage(err error) (title, hint string) {
	if err == nil {
		return "", ""
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		title = strings.TrimSpace(apiErr.Message)
		if title == "" {
			title = fmt.Sprintf("request failed HTTP %d", apiErr.StatusCode)
		}
		return title, strings.TrimSpace(apiErr.Action)
	}

	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		innerTitle, innerHint := UserMessage(unwrapped)
		outer := err.Error()
		inner := unwrapped.Error()
		if prefix := strings.TrimSuffix(outer, ": "+inner); prefix != outer && strings.TrimSpace(prefix) != "" {
			return strings.TrimSpace(prefix), joinHints(innerTitle, innerHint)
		}
		return UserMessage(unwrapped)
	}

	return strings.TrimSpace(err.Error()), ""
}

func joinHints(parts ...string) string {
	var kept []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n")
}

func TransportError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}

	switch {
	case isTimeout(err):
		rumptylog.Debug("API request timed out", "error", err)
		return errors.New("request timed out; check --api-url and that the API is reachable")
	case isConnectionRefused(err):
		rumptylog.Debug("API connection refused", "error", err)
		return errors.New("could not connect to the API; check --api-url")
	default:
		rumptylog.Debug("API transport error", "error", err)
		return errors.New("could not reach the Rumpty API; use --log-level=debug for details")
	}
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded")
}

func isConnectionRefused(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return strings.Contains(strings.ToLower(opErr.Err.Error()), "connection refused")
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused")
}
