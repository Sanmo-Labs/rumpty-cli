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
	ListAssetBuckets(ctx context.Context, workspace string) ([]AssetBucket, error)
	CreateAssetBucket(ctx context.Context, workspace string, req CreateAssetBucketRequest, idempotency string) (AssetBucketOperationResult, error)
	CreateAssetAccessKey(ctx context.Context, workspace, bucketUID string, req CreateAssetAccessKeyRequest) (CreateAssetAccessKeyResult, error)
	RevokeAssetAccessKey(ctx context.Context, workspace, keyUID string) error
	Logout(ctx context.Context) error
	LogoutAll(ctx context.Context) error
	StartDevice(ctx context.Context) (DeviceAuthStartResponse, error)
	PollDeviceToken(ctx context.Context, deviceCode string) (DeviceAuthPollResponse, error)
	IssueSSHCert(ctx context.Context, workspace string, req CertRequest) (CertResponse, error)
	AddSSHKey(ctx context.Context, workspace string, req AddSSHKeyRequest) (SSHKey, error)
	ListSSHKeys(ctx context.Context, workspace string) ([]SSHKey, error)
	DeleteSSHKey(ctx context.Context, workspace, uid string) error
	ListAPIKeys(ctx context.Context) ([]APIKey, error)
	RevokeAPIKey(ctx context.Context, uid string) error
	ListBucketObjects(ctx context.Context, workspace, bucketUID string) ([]AssetObject, error)
	DeleteBucketObject(ctx context.Context, workspace, objectUID string) error
	ListFirewallPolicies(ctx context.Context, workspace string) ([]FirewallPolicy, error)
	CreateFirewallPolicy(ctx context.Context, workspace string, req CreateFirewallPolicyRequest) (FirewallPolicy, error)
	DeleteFirewallPolicy(ctx context.Context, workspace, uid string) error
	AddFirewallRule(ctx context.Context, workspace, policyUID string, req *AddFirewallRuleRequest) (FirewallRuleResult, error)
	DeleteFirewallRule(ctx context.Context, workspace, policyUID, ruleUID string) error
	ListVolumes(ctx context.Context, workspace string) ([]Volume, error)
	CreateVolume(ctx context.Context, workspace string, req CreateVolumeRequest) (VolumeOperationResult, error)
	DeleteVolume(ctx context.Context, workspace, uid string) (VolumeOperationResult, error)
	ListWorkspaceMembers(ctx context.Context, workspaceUID, workspaceSlug string) ([]WorkspaceMember, error)
	InviteWorkspaceMember(ctx context.Context, workspaceUID, workspaceSlug string, req InviteWorkspaceMemberRequest) (WorkspaceInvitation, error)
	RemoveWorkspaceMember(ctx context.Context, workspaceUID, workspaceSlug, memberUID string) (WorkspaceMember, error)
	UpdateWorkspaceMemberRole(ctx context.Context, workspaceUID, workspaceSlug, memberUID, role string) (WorkspaceMember, error)
	ListClusters(ctx context.Context, workspace string) ([]Cluster, error)
	DeleteCluster(ctx context.Context, workspace, uid string) (ClusterOperationResult, error)
	AttachVolume(ctx context.Context, workspace, volumeUID, vmUID string) (VolumeOperationResult, error)
	DetachVolume(ctx context.Context, workspace, volumeUID string) (VolumeOperationResult, error)
	ListInvitations(ctx context.Context, workspaceUID, workspaceSlug string) ([]WorkspaceInvitation, error)
	RevokeInvitation(ctx context.Context, workspaceUID, workspaceSlug, invitationUID string) (WorkspaceInvitation, error)
	ListDatabases(ctx context.Context, workspace string) ([]Database, error)
	DeleteDatabase(ctx context.Context, workspace, dbUID, idempotency string) (DatabaseOperationResult, error)
}

var _ ClientAPI = (*Client)(nil)
