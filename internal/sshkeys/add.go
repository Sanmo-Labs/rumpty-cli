package sshkeys

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Add reads an SSH public key from path (default ~/.ssh/id_ed25519.pub),
// validates it, and adds it to the account. An empty name uses the key's
// comment, falling back to the filename.
func Add(ctx context.Context, rt *app.Runtime, path, name string) error {
	w := errOut(rt)
	spin := term.StartLoading(w, "Reading ssh key")
	pubKey, keyName, err := loadPublicKey(path, name)
	spin.Stop()
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin = term.StartLoading(w, "Adding ssh key "+keyName)
	added, err := rt.API().AddSSHKey(ctx, workspace, api.AddSSHKeyRequest{Name: keyName, PublicKey: pubKey})
	spin.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Added %s (%s)", added.Name, added.Fingerprint)))
	return nil
}

// loadPublicKey reads and validates a single-line OpenSSH public key.
func loadPublicKey(path, name string) (string, string, error) {
	if strings.TrimSpace(path) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("find home directory: %w", err)
		}
		path = filepath.Join(home, ".ssh", "id_ed25519.pub")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	line := firstNonEmptyLine(string(data))
	if line == "" {
		return "", "", fmt.Errorf("%s contains no public key", path)
	}
	if strings.HasPrefix(line, "-----BEGIN") {
		return "", "", fmt.Errorf("%s looks like a private key; pass a .pub public key instead", path)
	}

	_, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return "", "", fmt.Errorf("invalid ssh public key in %s: %w", path, err)
	}

	keyName := strings.TrimSpace(name)
	if keyName == "" {
		keyName = strings.TrimSpace(comment)
	}
	if keyName == "" {
		keyName = strings.TrimSuffix(filepath.Base(path), ".pub")
	}
	return line, keyName, nil
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
