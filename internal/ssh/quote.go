package ssh

import "strconv"

// shellQuote wraps s for use in an OpenSSH ProxyCommand string (parsed by the system shell on Unix).
func shellQuote(s string) string {
	return strconv.Quote(s)
}
