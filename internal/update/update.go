package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
)

const (
	EnvDisable    = "RUMPTY_NO_UPDATE_NOTIFIER"
	checkInterval = 24 * time.Hour
	fetchTimeout  = 2 * time.Second
	waitTimeout   = fetchTimeout + 500*time.Millisecond
	stateFileName = "update-check.json"
	installHint   = "curl -fsSL https://get.rumptycloud.com | sh"
)

var LatestReleaseURL = "https://api.github.com/repos/cloverworks-hq/rumpty-cli/releases/latest"

type state struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest,omitempty"`
}

// Start checks for a newer release at most once a day and returns a function
// that prints a one-line notice to w after that check finds one. The check runs in the
// background; the returned function waits for it for at most waitTimeout, long
// enough for a timed-out check to be recorded so it is not retried every run.
func Start(ctx context.Context, current string, w io.Writer, userAgent string) func() {
	if _, ok := parse(current); !ok {
		return func() {}
	}
	path, err := statePath()
	if err != nil {
		return func() {}
	}
	saved := load(path)
	if time.Since(saved.CheckedAt) < checkInterval {
		return func() {}
	}

	done := make(chan string, 1)
	go func() {
		latest := fetchLatest(ctx, userAgent)
		next := state{CheckedAt: time.Now().UTC(), Latest: saved.Latest}
		if latest != "" {
			next.Latest = latest
		}
		_ = save(path, next)
		done <- next.Latest
	}()
	return func() {
		select {
		case latest := <-done:
			notify(w, current, latest)
		case <-time.After(waitTimeout):
		}
	}
}

func notify(w io.Writer, current, latest string) {
	if !Newer(latest, current) {
		return
	}
	fmt.Fprintf(w, "\nA new rumpty release is available: %s → %s\nUpdate with: %s\n",
		strings.TrimPrefix(current, "v"), strings.TrimPrefix(latest, "v"), installHint)
}

func fetchLatest(ctx context.Context, userAgent string) string {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // fixed GitHub releases URL; only tests override it
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return ""
	}
	if _, ok := parse(release.TagName); !ok {
		return ""
	}
	return release.TagName
}

// Newer reports whether latest is a higher release than current. Both are
// MAJOR.MINOR.PATCH with an optional leading "v"; anything else is never newer.
func Newer(latest, current string) bool {
	l, ok := parse(latest)
	if !ok {
		return false
	}
	c, ok := parse(current)
	if !ok {
		return false
	}
	return slices.Compare(l[:], c[:]) > 0
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func statePath() (string, error) {
	dir, err := credentials.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

func load(path string) state {
	var s state
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func save(path string, s state) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return credentials.WriteSecretFile(path, append(data, '\n'))
}
