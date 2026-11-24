package billing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Download saves an invoice PDF to path, defaulting to the filename the
// server suggests. An existing file is never overwritten without force.
func Download(ctx context.Context, rt *app.Runtime, uid, path string, force bool) error {
	spin := term.StartLoading(errOut(rt), "Downloading invoice "+uid)
	pdf, err := rt.API().DownloadInvoicePDF(ctx, uid)
	spin.Stop()
	if err != nil {
		return err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		path = pdf.Filename
	}
	if path == "" {
		path = "invoice.pdf"
	}

	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists; pass --force to overwrite it", path)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, pdf.Content, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, "Saved invoice to "+path))
	return nil
}
