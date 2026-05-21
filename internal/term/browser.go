package term

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Browser opens a URL in the system browser.
type Browser interface {
	Open(url string) error
}

// SystemBrowser uses open / xdg-open / rundll32.
type SystemBrowser struct{}

func (SystemBrowser) Open(url string) error {
	ctx := context.Background()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

// OpenBrowser opens url with the default system browser.
func OpenBrowser(url string) error {
	return (SystemBrowser{}).Open(url)
}

// NoopBrowser does not open a URL (tests and --no-browser).
type NoopBrowser struct{}

func (NoopBrowser) Open(string) error { return nil }
