package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
)

func Logout(ctx context.Context, rt *app.Runtime) error {
	creds, err := credentials.Load()
	if err != nil {
		return err
	}

	token := strings.TrimSpace(creds.Token)
	if token == "" {
		token = strings.TrimSpace(rt.Config.Token)
	}
	if token != "" {
		apiURL := creds.APIURL
		if strings.TrimSpace(apiURL) == "" {
			apiURL = rt.Config.APIURL
		}
		if err := rt.APIWithToken(apiURL, token).Logout(ctx); err != nil {
			var apiErr *api.Error
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
				rumptylog.Warn("server logout failed", "err", err)
			}
		}
	}

	if err := credentials.Clear(); err != nil {
		return err
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Logged out.")
	return nil
}
