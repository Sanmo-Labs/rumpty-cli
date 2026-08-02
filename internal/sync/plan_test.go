package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func file(key string, size int64, mod time.Time) FileInfo {
	return FileInfo{Key: key, Size: size, ModTime: mod}
}

func TestPlanUpload(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	local := map[string]FileInfo{
		"unchanged.txt":    file("unchanged.txt", 10, earlier),
		"new.txt":          file("new.txt", 5, now),
		"resized.txt":      file("resized.txt", 20, earlier),
		"edited-later.txt": file("edited-later.txt", 10, now),
	}
	remote := map[string]FileInfo{
		"unchanged.txt":    file("unchanged.txt", 10, now),
		"resized.txt":      file("resized.txt", 15, now),
		"edited-later.txt": file("edited-later.txt", 10, earlier),
		"removed.txt":      file("removed.txt", 7, earlier),
	}

	plan := PlanUpload(local, remote, false)
	require.Len(t, plan.Actions, 3)
	assert.Equal(t, 1, plan.Unchanged)
	for _, action := range plan.Actions {
		assert.Equal(t, ActionUpload, action.Kind)
	}
	assert.Equal(t, "edited-later.txt", plan.Actions[0].Key)
	assert.Equal(t, "new.txt", plan.Actions[1].Key)
	assert.Equal(t, "resized.txt", plan.Actions[2].Key)
}

func TestPlanUploadDelete(t *testing.T) {
	now := time.Now()
	local := map[string]FileInfo{
		"keep.txt": file("keep.txt", 1, now),
	}
	remote := map[string]FileInfo{
		"keep.txt": file("keep.txt", 1, now.Add(time.Minute)),
		"gone.txt": file("gone.txt", 2, now),
	}

	plan := PlanUpload(local, remote, true)
	require.Len(t, plan.Actions, 1)
	assert.Equal(t, ActionDeleteRemote, plan.Actions[0].Kind)
	assert.Equal(t, "gone.txt", plan.Actions[0].Key)
	assert.Equal(t, 1, plan.Unchanged)
}

func TestPlanDownload(t *testing.T) {
	now := time.Now()
	remote := map[string]FileInfo{
		"same.txt":    file("same.txt", 3, now),
		"missing.txt": file("missing.txt", 4, now),
		"resized.txt": file("resized.txt", 9, now),
	}
	local := map[string]FileInfo{
		"same.txt":    file("same.txt", 3, now.Add(-time.Hour)),
		"resized.txt": file("resized.txt", 5, now),
		"extra.txt":   file("extra.txt", 1, now),
	}

	plan := PlanDownload(remote, local)
	require.Len(t, plan.Actions, 2)
	assert.Equal(t, 1, plan.Unchanged)
	assert.Equal(t, "missing.txt", plan.Actions[0].Key)
	assert.Equal(t, "resized.txt", plan.Actions[1].Key)
	for _, action := range plan.Actions {
		assert.Equal(t, ActionDownload, action.Kind)
	}
}

func TestApplyActions(t *testing.T) {
	now := time.Now()
	remote := map[string]FileInfo{
		"keep.txt": file("keep.txt", 1, now),
		"gone.txt": file("gone.txt", 2, now),
	}

	applyActions(remote, []Action{
		{Kind: ActionUpload, Key: "new.txt", Size: 5},
		{Kind: ActionUpload, Key: "keep.txt", Size: 9},
		{Kind: ActionDeleteRemote, Key: "gone.txt"},
	})

	require.Len(t, remote, 2)
	assert.Equal(t, int64(5), remote["new.txt"].Size)
	assert.Equal(t, int64(9), remote["keep.txt"].Size)
	assert.NotContains(t, remote, "gone.txt")
}

func TestScanLocal(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeFile(dir+"/a.txt", "hello"))
	require.NoError(t, writeFile(dir+"/nested/b.txt", "world!"))

	files, err := ScanLocal(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, int64(5), files["a.txt"].Size)
	assert.Equal(t, int64(6), files["nested/b.txt"].Size)
}
