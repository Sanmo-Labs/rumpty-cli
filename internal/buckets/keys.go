package buckets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Keys lists a bucket's S3 access keys.
func Keys(ctx context.Context, rt *app.Runtime, bucketRef string, format term.OutputFormat) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	spin := term.StartLoading(w, "Listing access keys")
	keys, err := bucketKeys(ctx, rt, bucketRef, workspace)
	spin.Stop()
	if err != nil {
		return err
	}

	out := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(out, keys)
	}
	if len(keys) == 0 {
		fmt.Fprintln(out, term.Muted(out, "No access keys yet. Create one with rumpty bucket keys create"))
		return nil
	}
	table := term.NewTable(out, "NAME", "ACCESS KEY ID", "PERMISSIONS", "STATUS")
	for i := range keys {
		k := &keys[i]
		table.Row(k.Name, k.AccessKeyID, strings.Join(k.Permissions, ","), k.Status)
	}
	return table.Render()
}

// CreateKey creates a bucket access key and prints the one-time secret once.
func CreateKey(ctx context.Context, rt *app.Runtime, bucketRef, name string, permissions []string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	spin := term.StartLoading(w, "Creating access key "+name)
	result, err := rt.API().CreateAssetAccessKey(ctx, workspace, bucket.UID, api.CreateAssetAccessKeyRequest{
		Name:        name,
		Permissions: permissions,
	})
	spin.Stop()
	if err != nil {
		return err
	}

	out := rt.Streams.Out
	fmt.Fprintln(out, term.Success(out, fmt.Sprintf("Created access key %s (access key id %s)", result.Name, result.AccessKeyID)))
	fmt.Fprintln(out, term.Muted(out, "secret_access_key: "+result.SecretAccessKey))
	fmt.Fprintln(out, term.Muted(out, "This secret is shown only once and cannot be retrieved again. Store it now."))
	if result.Endpoint != "" {
		fmt.Fprintln(out, term.Muted(out, fmt.Sprintf("Endpoint %s, region %s, bucket %s", result.Endpoint, result.Region, result.Bucket)))
	}
	if strings.TrimSpace(result.OperationID) != "" {
		spin = term.StartLoading(w, "Waiting for the key to become active")
		if err := waitKeyOperation(ctx, rt, workspace, result.OperationID); err != nil {
			return err
		}
		spin.Stop()
		fmt.Fprintln(out, term.Success(out, "The key is active"))
	}
	return nil
}

// waitKeyOperation polls the key's provisioning operation until it succeeds
// or fails.
func waitKeyOperation(ctx context.Context, rt *app.Runtime, workspace, operationID string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	deadline := time.Now().Add(time.Minute)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for the access key to become active; the secret is already saved above")
		}
		op, err := rt.API().GetOperation(ctx, workspace, operationID)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(op.Status)) {
		case "failed":
			return fmt.Errorf("access key provisioning failed")
		case "succeeded":
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RevokeKey revokes one access key of a bucket, resolved from the key's
// name, access key id or uid.
func RevokeKey(ctx context.Context, rt *app.Runtime, bucketRef, keyRef string, yes bool) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	spin := term.StartLoading(w, "Finding access key "+keyRef)
	keys, err := rt.API().ListAssetAccessKeys(ctx, workspace, bucket.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findAccessKey(keys, keyRef, bucket.Name)
	if err != nil {
		return err
	}

	if !yes {
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Revoke access key %s on %s? Anything still using it loses access immediately", target.Name, bucket.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Revoking "+target.Name)
	if err := rt.API().RevokeAssetAccessKey(ctx, workspace, target.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Revoked access key %s", target.Name)))
	return nil
}

// bucketKeys resolves a bucket and lists its access keys.
func bucketKeys(ctx context.Context, rt *app.Runtime, bucketRef, workspace string) ([]api.AssetAccessKey, error) {
	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return nil, err
	}
	return rt.API().ListAssetAccessKeys(ctx, workspace, bucket.UID)
}

// findAccessKey matches a key by uid, access key id or name.
func findAccessKey(keys []api.AssetAccessKey, ref, bucketName string) (api.AssetAccessKey, error) {
	if ref == "" {
		return api.AssetAccessKey{}, fmt.Errorf("no access key specified for bucket %s", bucketName)
	}
	for i := range keys {
		k := &keys[i]
		if strings.EqualFold(k.UID, ref) || strings.EqualFold(k.AccessKeyID, ref) {
			return *k, nil
		}
	}
	var matches []api.AssetAccessKey
	for i := range keys {
		if strings.EqualFold(keys[i].Name, ref) {
			matches = append(matches, keys[i])
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.AssetAccessKey{}, fmt.Errorf("access key %q not found on bucket %s (list them with rumpty bucket keys ls)", ref, bucketName)
	default:
		ids := make([]string, 0, len(matches))
		for i := range matches {
			ids = append(ids, matches[i].UID)
		}
		return api.AssetAccessKey{}, fmt.Errorf("access key name %q is ambiguous on bucket %s; %d keys match, use the key uid or access key id: %s", ref, bucketName, len(matches), strings.Join(ids, ", "))
	}
}
