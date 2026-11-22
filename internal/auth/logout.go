package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
	rumptylog "github.com/cloverworks-hq/rumpty-cli/internal/log"
	rumptyssh "github.com/cloverworks-hq/rumpty-cli/internal/ssh"
	rumptysync "github.com/cloverworks-hq/rumpty-cli/internal/sync"
)

// Logout stops sync daemons, revokes the bucket access keys sync created,
// ends the server session, and deletes every locally stored secret.
func Logout(ctx context.Context, rt *app.Runtime) error {
	creds, err := credentials.Load()
	if err != nil {
		return err
	}

	token := strings.TrimSpace(creds.Token)
	apiURL := config.SavedAPIURL(creds)
	if token == "" {
		token = strings.TrimSpace(rt.Config.Token)
		apiURL = rt.Config.APIURL
	}
	var client api.ClientAPI
	if token != "" {
		client = rt.APIWithToken(apiURL, token)
	}

	if err := rumptysync.StopAllDaemons(rt); err != nil {
		rumptylog.Warn("could not stop sync daemons", "err", err)
	}
	notRevoked, err := rumptysync.RevokeSavedKeys(ctx, client)
	if err != nil {
		return err
	}

	if client != nil {
		if err := client.Logout(ctx); err != nil {
			var apiErr *api.Error
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
				rumptylog.Warn("server logout failed", "err", err)
			}
		}
	}

	if err := rumptyssh.ClearCertCache(); err != nil {
		return err
	}
	if err := credentials.Clear(); err != nil {
		return err
	}

	if len(notRevoked) > 0 {
		fmt.Fprintln(rt.Streams.ErrOut, "Could not revoke these sync keys; revoke them in the console under the bucket's access keys:")
		for _, key := range notRevoked {
			fmt.Fprintf(rt.Streams.ErrOut, "  %s\n", key)
		}
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Logged out.")
	return nil
}
