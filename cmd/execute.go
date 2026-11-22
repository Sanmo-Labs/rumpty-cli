package commands

import (
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
	"github.com/cloverworks-hq/rumpty-cli/pkg/cmdutil"
)

func Execute() {
	streams := term.System()
	rt := &app.Runtime{
		Config:  &config.Config{},
		Streams: streams,
	}
	cmdutil.CheckErr(NewRoot(rt).Execute())
}
