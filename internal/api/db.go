package api

import (
	"context"
	"net/url"
)

// Database is a managed database in a workspace.
type Database struct {
	UID           string `json:"uid"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Engine        string `json:"engine"`
	Version       string `json:"version,omitempty"`
	Status        string `json:"status"`
	DisplayStatus string `json:"display_status"`
	PlanSlug      string `json:"plan_slug,omitempty"`
	ZoneSlug      string `json:"zone_slug"`
	StorageGiB    int    `json:"storage_gib"`
	VCPU          int    `json:"vcpu"`
	MemoryMiB     int    `json:"memory_mib"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	PublicAccess  bool   `json:"public_access"`
}

// DatabaseOperationResult is returned when a database operation is queued.
type DatabaseOperationResult struct {
	OperationID string   `json:"operation_id"`
	Database    Database `json:"database"`
}

func dbPath(uid string) string {
	return "/v1/databases/" + url.PathEscape(uid)
}

// ListDatabases returns the databases in a workspace.
func (c *Client) ListDatabases(ctx context.Context, workspace string) ([]Database, error) {
	var data []Database
	if err := c.getWithOptions(ctx, "/v1/databases", &data, requestOptions{
		headers: map[string]string{
			headerWorkspaceSlug: workspace,
		},
	}); err != nil {
		return nil, err
	}
	if data == nil {
		return []Database{}, nil
	}
	return data, nil
}

// DeleteDatabase queues deletion of a database. The database's snapshots are
// kept, detached from the database.
func (c *Client) DeleteDatabase(ctx context.Context, workspace, uid, idempotency string) (DatabaseOperationResult, error) {
	var data DatabaseOperationResult
	err := c.deleteWithOptions(ctx, dbPath(uid), &data, workspaceRequestOptions(workspace, idempotency))
	return data, err
}
