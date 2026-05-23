package ssh

import (
	"os"
	"strings"
)

func rsyncMissingOnRemote(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "rsync: command not found") ||
		strings.Contains(s, "rsync: not found") ||
		strings.Contains(s, ": rsync: not found") ||
		strings.Contains(s, ": rsync: command not found")
}

// copyRecursive returns true when scp needs -r. Uploads auto-detect local directories;
// downloads require -r because the remote path cannot be stat'd from here.
func copyRecursive(paths CopyPaths, flag bool) bool {
	if flag {
		return true
	}
	if !paths.Upload {
		return false
	}
	info, err := os.Stat(paths.Local)
	if err != nil {
		return false
	}
	return info.IsDir()
}
