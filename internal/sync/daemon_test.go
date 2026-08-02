package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemonID(t *testing.T) {
	id := daemonID("/home/u/photos", "my-backups", "")
	assert.Len(t, id, 12)
	assert.Equal(t, id, daemonID("/home/u/photos", "my-backups", ""))
	assert.NotEqual(t, id, daemonID("/home/u/photos", "my-backups", "sub"))
	assert.NotEqual(t, id, daemonID("/home/u/other", "my-backups", ""))
}

func TestDaemonStateRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	now := time.Now().UTC().Truncate(time.Second)
	state := &DaemonState{
		ID:         daemonID("/tmp/x", "b", "p"),
		PID:        1234,
		LocalPath:  "/tmp/x",
		Bucket:     "b",
		Prefix:     "p",
		Workspace:  "ws",
		StartedAt:  now,
		LastSyncAt: &now,
		LastError:  "boom",
		LogFile:    "/tmp/x.log",
	}
	require.NoError(t, writeState(state))

	got, err := readState(state.ID)
	require.NoError(t, err)
	assert.Equal(t, state, got)

	states, err := listStates()
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, state.ID, states[0].ID)

	removeState(state.ID)
	_, err = readState(state.ID)
	assert.Error(t, err)
}

func TestHumanAgo(t *testing.T) {
	assert.Equal(t, "never", humanAgo(nil))
	recent := time.Now().Add(-30 * time.Second)
	assert.Equal(t, "30s ago", humanAgo(&recent))
	older := time.Now().Add(-3 * time.Hour)
	assert.Equal(t, "3h ago", humanAgo(&older))
}
