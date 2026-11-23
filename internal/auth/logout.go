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
	client, _, err := savedSessionClient(rt)
	if err != nil {
		return err
	}
	notRevoked, err := stopSync(ctx, rt, client)
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
	if err := clearLocal(rt, notRevoked); err != nil {
		return err
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Logged out.")
	return nil
}

// LogoutAll ends every session on the account, then removes the local one.
// Sync keys are revoked first, while the session can still do it. If the
// server does not confirm, the local login is kept so the user can retry.
func LogoutAll(ctx context.Context, rt *app.Runtime) error {
	client, token, err := savedSessionClient(rt)
	if err != nil {
		return err
	}
	if client == nil {
		return errors.New("not logged in: run rumpty login, then rumpty logout --all")
	}
	if strings.HasPrefix(token, "rumpty_") {
		return errors.New("signing out of all sessions needs a browser login, not an API key: run rumpty login, then rumpty logout --all")
	}
	notRevoked, err := stopSync(ctx, rt, client)
	if err != nil {
		return err
	}
	if err := client.LogoutAll(ctx); err != nil {
		printNotRevoked(rt, notRevoked)
		reason := fmt.Sprintf("could not sign out of all sessions: %v", err)
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case http.StatusUnauthorized:
				reason = "your session has already ended, so other sessions were not signed out; run rumpty login, then rumpty logout --all"
			case http.StatusForbidden:
				reason = "signing out of all sessions needs a browser login, not an API key"
			case http.StatusNotFound:
				reason = "this Rumpty API does not support signing out of all sessions yet"
			}
		}
		return fmt.Errorf("%s. Sync daemons were stopped and their keys revoked; you are still logged in here", reason)
	}
	if err := clearLocal(rt, notRevoked); err != nil {
		return err
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Signed out of all sessions.")
	return nil
}

func savedSessionClient(rt *app.Runtime) (api.ClientAPI, string, error) {
	creds, err := credentials.Load()
	if err != nil {
		return nil, "", err
	}
	token := strings.TrimSpace(creds.Token)
	apiURL := config.SavedAPIURL(creds)
	if token == "" {
		token = strings.TrimSpace(rt.Config.Token)
		apiURL = rt.Config.APIURL
	}
	if token == "" {
		return nil, "", nil
	}
	return rt.APIWithToken(apiURL, token), token, nil
}

func stopSync(ctx context.Context, rt *app.Runtime, client api.ClientAPI) ([]string, error) {
	if err := rumptysync.StopAllDaemons(rt); err != nil {
		rumptylog.Warn("could not stop sync daemons", "err", err)
	}
	return rumptysync.RevokeSavedKeys(ctx, client)
}

func clearLocal(rt *app.Runtime, notRevoked []string) error {
	if err := rumptyssh.ClearCertCache(); err != nil {
		return err
	}
	if err := credentials.Clear(); err != nil {
		return err
	}
	printNotRevoked(rt, notRevoked)
	return nil
}

func printNotRevoked(rt *app.Runtime, notRevoked []string) {
	if len(notRevoked) == 0 {
		return
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Could not revoke these sync keys; revoke them in the console under the bucket's access keys:")
	for _, key := range notRevoked {
		fmt.Fprintf(rt.Streams.ErrOut, "  %s\n", key)
	}
}
