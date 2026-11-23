package update_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/update"
)

func TestNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.1.0", "0.0.13", true},
		{"v0.0.13", "0.0.12", true},
		{"v1.0.0", "0.99.99", true},
		{"v0.0.12", "0.0.12", false},
		{"v0.0.11", "0.0.12", false},
		{"v0.0.13", "dev", false},
		{"", "0.0.12", false},
		{"v0.1.0-rc.1", "0.0.12", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, update.Newer(tt.latest, tt.current), "%s vs %s", tt.latest, tt.current)
	}
}

func fakeGitHub(t *testing.T, tag string, delay time.Duration) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(delay)
		if tag == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	prev := update.LatestReleaseURL
	update.LatestReleaseURL = srv.URL
	t.Cleanup(func() { update.LatestReleaseURL = prev })
	return &hits
}

func TestStart_notifiesAndChecksOncePerDay(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	hits := fakeGitHub(t, "v0.1.0", 0)

	var out bytes.Buffer
	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Contains(t, out.String(), "0.0.12 → 0.1.0")
	assert.Contains(t, out.String(), "get.rumptycloud.com")

	out.Reset()
	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Empty(t, out.String())
	assert.Equal(t, int32(1), hits.Load())

	info, err := os.Stat(filepath.Join(dir, "rumpty", "update-check.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestStart_silentWhenCurrent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fakeGitHub(t, "v0.0.12", 0)
	var out bytes.Buffer
	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Empty(t, out.String())
}

func TestStart_failedCheckIsNotRetriedEveryRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hits := fakeGitHub(t, "", 0)
	var out bytes.Buffer
	update.Start(context.Background(), "0.0.12", &out, "test")()
	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Empty(t, out.String())
	assert.Equal(t, int32(1), hits.Load())
}

func TestStart_devBuildNeverChecks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hits := fakeGitHub(t, "v0.1.0", 0)
	var out bytes.Buffer
	update.Start(context.Background(), "dev", &out, "test")()
	assert.Empty(t, out.String())
	assert.Equal(t, int32(0), hits.Load())
}

func TestStart_slowGitHubIsBoundedAndRecorded(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hits := fakeGitHub(t, "v0.1.0", 3*time.Second)
	var out bytes.Buffer
	begin := time.Now()
	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Less(t, time.Since(begin), 3*time.Second)
	assert.Empty(t, out.String())

	update.Start(context.Background(), "0.0.12", &out, "test")()
	assert.Equal(t, int32(1), hits.Load())
}
