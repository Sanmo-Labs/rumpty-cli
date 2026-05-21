package commands

import (
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
	"github.com/Sanmo-Labs/rumpty-cli/pkg/cmdutil"
)

func Execute() {
	streams := term.System()
	rt := &app.Runtime{
		Config:  &config.Config{},
		Streams: streams,
	}
	cmdutil.CheckErr(NewRoot(rt).Execute())
}
