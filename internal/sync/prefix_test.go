package sync

import "testing"

func TestJoinPrefix(t *testing.T) {
	cases := []struct {
		name string
		base string
		user string
		want string
	}{
		{"both empty", "", "", ""},
		{"base only", "projects/019e/sync-test-bucket", "", "projects/019e/sync-test-bucket"},
		{"user only", "", "logs", "logs"},
		{"both", "projects/019e/sync-test-bucket", "logs", "projects/019e/sync-test-bucket/logs"},
		{"trims slashes", "/projects/019e/", "/logs/", "projects/019e/logs"},
		{"trims spaces", "  projects/019e  ", "  logs  ", "projects/019e/logs"},
		{"user with subpath", "base", "a/b", "base/a/b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := joinPrefix(tc.base, tc.user); got != tc.want {
				t.Fatalf("joinPrefix(%q, %q) = %q, want %q", tc.base, tc.user, got, tc.want)
			}
		})
	}
}
