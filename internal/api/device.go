package api

import "context"

type DeviceAuthStartResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceAuthTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

const DeviceAuthStatusPending = "authorization_pending"

type DeviceAuthPollResponse struct {
	Status   string `json:"status,omitempty"`
	Token    string `json:"token,omitempty"`
	User     User   `json:"user,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

func (c *Client) StartDevice(ctx context.Context) (DeviceAuthStartResponse, error) {
	var data DeviceAuthStartResponse
	if err := c.post(ctx, "/v1/auth/device", struct{}{}, &data, requestOptions{}); err != nil {
		return DeviceAuthStartResponse{}, err
	}
	return data, nil
}

func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (DeviceAuthPollResponse, error) {
	var data DeviceAuthPollResponse
	if err := c.post(ctx, "/v1/auth/device/token", DeviceAuthTokenRequest{DeviceCode: deviceCode}, &data, requestOptions{}); err != nil {
		return DeviceAuthPollResponse{}, err
	}
	return data, nil
}
