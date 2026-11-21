package ssh

import (
	"runtime"
	"strings"
)

// shellQuote quotes s for the shell that runs an OpenSSH ProxyCommand:
// /bin/sh on Unix, the Windows command-line parser on Windows.
func shellQuote(s string) string {
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return posixQuote(s)
}

// posixQuote single-quotes s so /bin/sh passes it through unchanged.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
