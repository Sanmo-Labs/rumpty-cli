package api

import "context"

// ClientAPI is the HTTP API surface used by commands. *Client implements it.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=mocks/mock_client.go -package=mocks . ClientAPI
type ClientAPI interface {
	Me(ctx context.Context) (User, error)
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
	Logout(ctx context.Context) error
	StartDevice(ctx context.Context) (DeviceAuthStartResponse, error)
	PollDeviceToken(ctx context.Context, deviceCode string) (DeviceAuthPollResponse, error)
	IssueSSHCert(ctx context.Context, workspace string, req CertRequest) (CertResponse, error)
}

var _ ClientAPI = (*Client)(nil)
