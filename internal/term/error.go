package term

import (
	"fmt"
	"io"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func PrintError(w io.Writer, err error) {
	title, hint := api.UserMessage(err)
	if title == "" {
		title = "something went wrong"
	}
	_, _ = fmt.Fprintf(w, "%s\n", Error(w, title))
	for line := range strings.SplitSeq(strings.TrimSpace(hint), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		_, _ = fmt.Fprintf(w, "%s\n", Muted(w, "  "+line))
	}
}
