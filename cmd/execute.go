package commands

import (
	"context"
	"os"

	xterm "golang.org/x/term"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
	"github.com/cloverworks-hq/rumpty-cli/internal/update"
	"github.com/cloverworks-hq/rumpty-cli/internal/version"
	"github.com/cloverworks-hq/rumpty-cli/pkg/cmdutil"
)

func Execute() {
	streams := term.System()
	rt := &app.Runtime{
		Config:  &config.Config{},
		Streams: streams,
	}
	notify := func() {}
	if updateCheckWanted(os.Args[1:]) {
		notify = update.Start(context.Background(), version.Version, streams.ErrOut, api.UserAgent())
	}
	err := NewRoot(rt).Execute()
	notify()
	cmdutil.CheckErr(err)
}

func updateCheckWanted(args []string) bool {
	if os.Getenv(update.EnvDisable) != "" || os.Getenv("CI") != "" {
		return false
	}
	if !xterm.IsTerminal(int(os.Stderr.Fd())) { //nolint:gosec // file descriptors are small on supported platforms
		return false
	}
	if len(args) > 0 {
		switch args[0] {
		case "completion", "__complete", "__completeNoDesc":
			return false
		}
	}
	return true
}
