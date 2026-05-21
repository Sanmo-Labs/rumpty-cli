package term

import (
	"io"
	"strings"
)

const (
	rumptyLogotype = `▄▖       ▗   ▄▖▜      ▌
▙▘▌▌▛▛▌▛▌▜▘▌▌▌ ▐ ▛▌▌▌▛▌
▌▌▙▌▌▌▌▙▌▐▖▙▌▙▖▐▖▙▌▙▌▙▌
       ▌   ▄▌          `
	rumptyBannerFooter = "RumptyCloud\nSanmọ̀Labs™ - Surpass your limits!"
)

func PrintBanner(w io.Writer) {
	_, _ = io.WriteString(w, strings.TrimRight(rumptyLogotype, "\n"))
	_, _ = io.WriteString(w, "\n\n")
	_, _ = io.WriteString(w, rumptyBannerFooter)
	_, _ = io.WriteString(w, "\n")
}
