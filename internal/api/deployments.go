package api

import (
	"context"
	"net/url"
	"strconv"
)

// Deployment is one RumptyCloud deployment as the API returns it.
type Deployment struct {
	UID                    string `json:"uid"`
	Name                   string `json:"name"`
	Slug                   string `json:"slug"`
	Status                 string `json:"status"`
	DisplayStatus          string `json:"display_status,omitempty"`
	PlanSlug               string `json:"plan_slug"`
	PlanName               string `json:"plan_name,omitempty"`
	CPULimitMillis         int64  `json:"cpu_limit_millis,omitempty"`
	MemoryMiB              int64  `json:"memory_mib,omitempty"`
	MonthlyPriceCents      int64  `json:"monthly_price_cents,omitempty"`
	Currency               string `json:"currency,omitempty"`
	RepoURL                string `json:"repo_url"`
	GitProvider            string `json:"git_provider,omitempty"`
	GitRepoFullName        string `json:"git_repo_full_name,omitempty"`
	Branch                 string `json:"branch"`
	RootDirectory          string `json:"root_directory,omitempty"`
	AppType                string `json:"app_type"`
	BuildMethod            string `json:"build_method"`
	BuildCommand           string `json:"build_command,omitempty"`
	StartCommand           string `json:"start_command,omitempty"`
	HealthCheckPath        string `json:"health_check_path,omitempty"`
	Port                   int    `json:"port"`
	AutoDeployEnabled      bool   `json:"auto_deploy_enabled"`
	SecurityScanEnabled    bool   `json:"security_scan_enabled"`
	PreviewAppsEnabled     bool   `json:"preview_apps_enabled"`
	ArtifactRetention      int    `json:"artifact_retention"`
	VolumeMountPath        string `json:"volume_mount_path,omitempty"`
	VolumeSizeGiB          int    `json:"volume_size_gib,omitempty"`
	ScaleToZeroEnabled     bool   `json:"scale_to_zero_enabled,omitempty"`
	MinReplicas            int    `json:"min_replicas,omitempty"`
	MaxReplicas            int    `json:"max_replicas,omitempty"`
	CurrentReplicas        int    `json:"current_replicas,omitempty"`
	AppUID                 string `json:"app_uid,omitempty"`
	CustomDomain           string `json:"custom_domain,omitempty"`
	LastRunUID             string `json:"last_run_uid,omitempty"`
	LastRunStatus          string `json:"last_run_status,omitempty"`
	PublicURL              string `json:"public_url,omitempty"`
	DefaultPublicURL       string `json:"default_public_url,omitempty"`
	CreatedAt              string `json:"created_at"`
	LastFailure            string `json:"last_failure,omitempty"`
	EnvChangesPending      bool   `json:"env_changes_pending,omitempty"`
	EnvChangesRequireBuild bool   `json:"env_changes_require_build,omitempty"`
}

// ListDeployments returns the workspace's deployments, newest first.
// Deleted deployments, pull request previews and one-click app components
// are not included.
func (c *Client) ListDeployments(ctx context.Context, workspace string) ([]Deployment, error) {
	var data []Deployment
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	if err := c.getWithOptions(ctx, "/v1/deployments", &data, opts); err != nil {
		return nil, err
	}
	if data == nil {
		return []Deployment{}, nil
	}
	return data, nil
}

// GetDeployment returns one deployment by uid.
func (c *Client) GetDeployment(ctx context.Context, workspace, uid string) (Deployment, error) {
	var data Deployment
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	if err := c.getWithOptions(ctx, "/v1/deployments/"+url.PathEscape(uid), &data, opts); err != nil {
		return Deployment{}, err
	}
	return data, nil
}

// ActivityDay is one day of a deployment's run counts. Days are calendar
// dates in West Africa Time (UTC+1).
type ActivityDay struct {
	Date      string `json:"date"`
	Count     int    `json:"count"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
}

// ListDeploymentActivity returns how many runs the deployment started per
// day, with how many succeeded and failed. Only days with at least one run
// are included, oldest first. days is how far back to count, up to 366.
func (c *Client) ListDeploymentActivity(ctx context.Context, workspace, uid string, days int) ([]ActivityDay, error) {
	var data []ActivityDay
	query := url.Values{}
	if days > 0 {
		query.Set("days", strconv.Itoa(days))
	}
	path := "/v1/deployments/" + url.PathEscape(uid) + "/activity"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	if err := c.getWithOptions(ctx, path, &data, opts); err != nil {
		return nil, err
	}
	if data == nil {
		return []ActivityDay{}, nil
	}
	return data, nil
}

// DeploymentRun is one build or rollout of a deployment.
type DeploymentRun struct {
	UID           string `json:"uid"`
	DeploymentUID string `json:"deployment_uid"`
	OperationID   string `json:"operation_id"`
	Trigger       string `json:"trigger"`
	Status        string `json:"status"`
	Branch        string `json:"branch"`
	CommitSHA     string `json:"commit_sha,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	TriggerActor  string `json:"trigger_actor,omitempty"`
}

// ListDeploymentRuns returns the deployment's runs, newest first. pageSize
// of 0 fetches the API default of 10; page is the 1-based page number.
func (c *Client) ListDeploymentRuns(ctx context.Context, workspace, uid string, page, pageSize int) ([]DeploymentRun, error) {
	var data []DeploymentRun
	query := url.Values{}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	path := "/v1/deployments/" + url.PathEscape(uid) + "/runs"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	opts := requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}
	if err := c.getWithOptions(ctx, path, &data, opts); err != nil {
		return nil, err
	}
	if data == nil {
		return []DeploymentRun{}, nil
	}
	return data, nil
}
