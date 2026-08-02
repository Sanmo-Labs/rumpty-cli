package sync

import (
	"path"
	"strings"
)

type Filter struct {
	Include []string
	Exclude []string
}

func (f Filter) Empty() bool {
	return len(f.Include) == 0 && len(f.Exclude) == 0
}

// Match reports whether key passes the filter: when include patterns exist the
// key must match one of them, and it must match no exclude pattern.
func (f Filter) Match(key string) bool {
	if len(f.Include) > 0 && !matchAny(f.Include, key) {
		return false
	}
	return !matchAny(f.Exclude, key)
}

func (f Filter) Apply(files map[string]FileInfo) map[string]FileInfo {
	if f.Empty() {
		return files
	}
	out := make(map[string]FileInfo, len(files))
	for key, info := range files {
		if f.Match(key) {
			out[key] = info
		}
	}
	return out
}

func matchAny(patterns []string, key string) bool {
	for _, pattern := range patterns {
		if matchPattern(pattern, key) {
			return true
		}
	}
	return false
}

// matchPattern matches pattern against the slash-relative key. Patterns
// without a slash also match against the base name so "*.jpg" applies at any
// depth, and a trailing "/**" matches everything under a directory.
func matchPattern(pattern, key string) bool {
	pattern = strings.Trim(strings.TrimSpace(pattern), "/")
	if pattern == "" {
		return false
	}
	if ok, _ := path.Match(pattern, key); ok {
		return true
	}
	if !strings.Contains(pattern, "/") {
		if ok, _ := path.Match(pattern, path.Base(key)); ok {
			return true
		}
	}
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(key, strings.TrimSuffix(pattern, "**"))
	}
	return false
}
