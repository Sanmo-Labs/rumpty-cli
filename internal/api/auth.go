package api

import (
	"context"
	"fmt"
)

func (c *Client) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	var data AuthResponse
	if err := c.post(ctx, "/v1/auth/login", req, &data, requestOptions{}); err != nil {
		return AuthResponse{}, err
	}
	return data, nil
}

func (c *Client) VerifyLoginOTP(ctx context.Context, req VerifyLoginOTPRequest) (AuthResponse, error) {
	var data AuthResponse
	if err := c.post(ctx, "/v1/auth/login/verify", req, &data, requestOptions{}); err != nil {
		return AuthResponse{}, err
	}
	if data.Token == "" {
		return AuthResponse{}, fmt.Errorf("login response did not include a token")
	}
	return data, nil
}

func (c *Client) ResendLoginOTP(ctx context.Context, otpSession string) error {
	return c.post(ctx, "/v1/auth/login/resend", ResendLoginOTPRequest{OTPSession: otpSession}, nil, requestOptions{})
}

func (c *Client) Logout(ctx context.Context) error {
	return c.post(ctx, "/v1/auth/logout", struct{}{}, nil, requestOptions{})
}

func (c *Client) Me(ctx context.Context) (User, error) {
	var data User
	if err := c.get(ctx, "/v1/me", &data); err != nil {
		return User{}, err
	}
	return data, nil
}
