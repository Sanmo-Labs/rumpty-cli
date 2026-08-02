package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitBucketArg(t *testing.T) {
	tests := []struct {
		arg    string
		name   string
		prefix string
	}{
		{"my-backups", "my-backups", ""},
		{"my-backups/photos", "my-backups", "photos"},
		{"my-backups/photos/2026", "my-backups", "photos/2026"},
		{"/my-backups/photos/", "my-backups", "photos"},
		{"", "", ""},
	}
	for _, tt := range tests {
		name, prefix := splitBucketArg(tt.arg)
		assert.Equal(t, tt.name, name, tt.arg)
		assert.Equal(t, tt.prefix, prefix, tt.arg)
	}
}

func TestCleanPatterns(t *testing.T) {
	assert.Equal(t, []string{"*.jpg", "*.png"}, cleanPatterns([]string{"*.jpg", " *.png"}))
	assert.Equal(t, []string{"*.log"}, cleanPatterns([]string{" ", "*.log", ""}))
	assert.Empty(t, cleanPatterns(nil))
}

func TestDefaultBucketName(t *testing.T) {
	assert.Equal(t, "photos", defaultBucketName("./photos"))
	assert.Equal(t, "app", defaultBucketName("projects/app/"))
	assert.Empty(t, defaultBucketName("/"))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(cwd), defaultBucketName("."))

	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	assert.Equal(t, filepath.Base(dir), defaultBucketName(path))
}
