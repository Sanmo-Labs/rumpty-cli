package cmdutil

import (
	"fmt"
	"os"

	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
)

func CheckErr(err error) {
	if err == nil {
		return
	}
	if config.IsUsageError(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "rumpty: %v\n", err)
	os.Exit(1)
}
