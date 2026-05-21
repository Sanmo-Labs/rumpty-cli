package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

const errLoginTimedOut = "login timed out: run rumpty login again"

type LoginOptions struct {
	NoBrowser bool
}

func Login(ctx context.Context, rt *app.Runtime, apiKey string, opts LoginOptions) error {
	if strings.TrimSpace(apiKey) != "" {
		return loginWithAPIKey(ctx, rt, strings.TrimSpace(apiKey))
	}
	return loginDevice(ctx, rt, opts.NoBrowser)
}

func loginWithAPIKey(ctx context.Context, rt *app.Runtime, apiKey string) error {
	user, err := rt.APIWithToken(rt.Config.APIURL, apiKey).Me(ctx)
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.Message != "" {
			return errors.New(apiErr.Message)
		}
		return errors.New("API key is invalid or expired")
	}
	return SaveSession(rt, rt.Config.APIURL, apiKey, user.Username)
}

func loginDevice(ctx context.Context, rt *app.Runtime, noBrowser bool) error {
	if ok, err := AlreadyLoggedIn(ctx, rt); err != nil {
		return err
	} else if ok {
		return nil
	}

	rumptylog.Debug("starting device authorization", "api_url", rt.Config.APIURL)
	client := rt.APIWithToken(rt.Config.APIURL, "")
	start, err := client.StartDevice(ctx)
	if err != nil {
		return err
	}
	if start.DeviceCode == "" || start.UserCode == "" {
		return errors.New("device authorization response is incomplete")
	}

	interval := start.Interval
	if interval <= 0 {
		interval = 5
	}
	deadline := time.Now().Add(time.Duration(start.ExpiresIn) * time.Second)

	fmt.Fprintln(rt.Streams.ErrOut, "Authenticate Rumpty in your browser.")
	fmt.Fprintf(rt.Streams.ErrOut, "\nYour one-time code is %s.\n\n", start.UserCode)
	openURL := start.VerificationURIComplete
	if openURL == "" {
		openURL = start.VerificationURI
	}
	fmt.Fprintf(rt.Streams.ErrOut, "Open this URL in your browser.\n  %s\n\n", openURL)
	if _, err := term.Line(rt.Streams.In, rt.Streams.ErrOut, "Press Enter to open the browser. You can also open the URL manually.", ""); err != nil {
		return err
	}
	if openURL != "" && !noBrowser {
		if err := rt.OpenBrowser(openURL); err != nil {
			fmt.Fprintf(rt.Streams.ErrOut, "Could not open the browser. %v\n", err)
		}
	}
	fmt.Fprintln(rt.Streams.ErrOut, "Waiting for authorization...")

	for time.Now().Before(deadline) {
		rumptylog.Debug("polling device authorization")
		poll, err := client.PollDeviceToken(ctx, start.DeviceCode)
		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.Message), "expired") {
				return errors.New(errLoginTimedOut)
			}
			return err
		}
		if poll.Token != "" {
			return SaveSession(rt, rt.Config.APIURL, poll.Token, poll.User.Username)
		}
		if poll.Status != api.DeviceAuthStatusPending && poll.Status != "" {
			return fmt.Errorf("unexpected authorization status %q", poll.Status)
		}
		sleep := interval
		if poll.Interval > 0 {
			sleep = poll.Interval
		}
		time.Sleep(time.Duration(sleep) * time.Second)
	}
	return errors.New(errLoginTimedOut)
}

func AlreadyLoggedIn(ctx context.Context, rt *app.Runtime) (bool, error) {
	creds, err := credentials.Load()
	if err != nil {
		return false, err
	}
	token := strings.TrimSpace(creds.Token)
	if token == "" {
		return false, nil
	}
	apiURL := rt.Config.APIURL
	if strings.TrimSpace(apiURL) == "" {
		apiURL = creds.APIURL
	}
	user, err := rt.APIWithToken(apiURL, token).Me(ctx)
	if err != nil {
		return false, nil
	}
	name := user.Username
	if name == "" {
		name = creds.Username
	}
	fmt.Fprintf(rt.Streams.ErrOut, "Already logged in as %s.\n", name)
	return true, nil
}

func SaveSession(rt *app.Runtime, apiURL, token, username string) error {
	if token == "" {
		return errors.New("token is required")
	}
	if err := credentials.Save(credentials.File{
		APIURL:   apiURL,
		Token:    token,
		Username: username,
	}); err != nil {
		return err
	}
	if username != "" {
		fmt.Fprintf(rt.Streams.ErrOut, "Logged in as %s.\n", username)
	} else {
		fmt.Fprintln(rt.Streams.ErrOut, "Logged in.")
	}
	return nil
}
