package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterMatch(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		key    string
		want   bool
	}{
		{"empty filter allows all", Filter{}, "a/b/c.txt", true},
		{"include extension at depth", Filter{Include: []string{"*.jpg"}}, "photos/2026/beach.jpg", true},
		{"include extension no match", Filter{Include: []string{"*.jpg"}}, "notes.txt", false},
		{"include multiple", Filter{Include: []string{"*.jpg", "*.png"}}, "icon.png", true},
		{"include exact name", Filter{Include: []string{"report.pdf"}}, "report.pdf", true},
		{"include exact name nested", Filter{Include: []string{"report.pdf"}}, "docs/report.pdf", true},
		{"exclude extension", Filter{Exclude: []string{"*.log"}}, "app/server.log", false},
		{"exclude directory tree", Filter{Exclude: []string{"node_modules/**"}}, "node_modules/pkg/index.js", false},
		{"exclude directory tree misses sibling", Filter{Exclude: []string{"node_modules/**"}}, "src/index.js", true},
		{"include then exclude wins", Filter{Include: []string{"*.js"}, Exclude: []string{"dist/**"}}, "dist/bundle.js", false},
		{"path pattern", Filter{Include: []string{"docs/*.md"}}, "docs/readme.md", true},
		{"path pattern wrong depth", Filter{Include: []string{"docs/*.md"}}, "docs/sub/readme.md", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.filter.Match(tt.key), tt.name)
	}
}

func TestFilterApply(t *testing.T) {
	now := time.Now()
	files := map[string]FileInfo{
		"a.jpg":     file("a.jpg", 1, now),
		"b.txt":     file("b.txt", 2, now),
		"sub/c.jpg": file("sub/c.jpg", 3, now),
	}

	got := Filter{Include: []string{"*.jpg"}}.Apply(files)
	assert.Len(t, got, 2)
	assert.Contains(t, got, "a.jpg")
	assert.Contains(t, got, "sub/c.jpg")

	same := Filter{}.Apply(files)
	assert.Len(t, same, 3)
}
