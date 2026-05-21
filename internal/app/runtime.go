package app

import (
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

type Runtime struct {
	Config    *config.Config
	Streams   term.Streams
	APIClient api.ClientAPI // optional; used in tests
	Browser   term.Browser  // optional; nil uses term.SystemBrowser
}

func (rt *Runtime) OpenBrowser(url string) error {
	if url == "" {
		return nil
	}
	b := rt.Browser
	if b == nil {
		b = term.SystemBrowser{}
	}
	return b.Open(url)
}

func (rt *Runtime) APIWithToken(apiURL, token string) api.ClientAPI {
	if rt.APIClient != nil {
		return rt.APIClient
	}
	if strings.TrimSpace(apiURL) == "" {
		apiURL = rt.Config.APIURL
	}
	return api.NewClient(apiURL, token)
}

func (rt *Runtime) API() api.ClientAPI {
	return rt.APIWithToken("", rt.Config.Token)
}
