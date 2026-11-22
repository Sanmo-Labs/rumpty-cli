package cmdutil

import (
	"fmt"
	"os"

	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func CheckErr(err error) {
	if err == nil {
		return
	}
	if config.IsUsageError(err) {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	term.PrintError(os.Stderr, err)
	os.Exit(1)
}
