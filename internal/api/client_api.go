package api

import "context"

// ClientAPI is the HTTP API surface used by commands. *Client implements it.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=mocks/mock_client.go -package=mocks . ClientAPI
type ClientAPI interface {
	Me(ctx context.Context) (User, error)
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
	ListVMs(ctx context.Context, workspace string) ([]VM, error)
	ListVMApps(ctx context.Context, workspace, vmUID string) ([]VMApp, error)
	StartVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error)
	StopVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error)
	RebootVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error)
	DeleteVM(ctx context.Context, workspace, vmUID, idempotency string) (VMOperationResult, error)
	ExposeVMApp(ctx context.Context, workspace, vmUID string, req ExposeVMAppRequest, idempotency string) (ExposeVMAppResult, error)
	UnexposeVMApp(ctx context.Context, workspace, vmUID, app string, idempotency string) (UnexposeVMAppResult, error)
	GetOperation(ctx context.Context, workspace, operationID string) (Operation, error)
	Logout(ctx context.Context) error
	StartDevice(ctx context.Context) (DeviceAuthStartResponse, error)
	PollDeviceToken(ctx context.Context, deviceCode string) (DeviceAuthPollResponse, error)
	IssueSSHCert(ctx context.Context, workspace string, req CertRequest) (CertResponse, error)
}

var _ ClientAPI = (*Client)(nil)
