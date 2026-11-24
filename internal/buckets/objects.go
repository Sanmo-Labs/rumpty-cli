package buckets

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// findBucket resolves a bucket by uid, name or slug in the workspace.
func findBucket(buckets []api.AssetBucket, ref, workspaceSlug string) (api.AssetBucket, error) {
	for i := range buckets {
		b := &buckets[i]
		if b.UID == ref || b.Name == ref || b.Slug == ref {
			return *b, nil
		}
	}
	return api.AssetBucket{}, fmt.Errorf("no bucket %q in workspace %s", ref, workspaceSlug)
}

func resolveBucket(ctx context.Context, rt *app.Runtime, ref, workspace string) (api.AssetBucket, error) {
	spin := term.StartLoading(errOut(rt), "Finding bucket")
	buckets, err := rt.API().ListAssetBuckets(ctx, workspace)
	spin.Stop()
	if err != nil {
		return api.AssetBucket{}, err
	}
	return findBucket(buckets, ref, workspace)
}

// Objects lists a bucket's objects, optionally filtered by key prefix.
func Objects(ctx context.Context, rt *app.Runtime, bucketRef, prefix string, format term.OutputFormat) error {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	spin := term.StartLoading(errOut(rt), "Listing objects in "+bucket.Name)
	objects, err := rt.API().ListBucketObjects(ctx, workspace, bucket.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	if strings.TrimSpace(prefix) != "" {
		filtered := make([]api.AssetObject, 0, len(objects))
		for i := range objects {
			if strings.HasPrefix(objects[i].Key, prefix) {
				filtered = append(filtered, objects[i])
			}
		}
		objects = filtered
	}

	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, objects)
	}
	if len(objects) == 0 {
		fmt.Fprintln(w, "No objects.")
		return nil
	}

	table := term.NewTable(w, "KEY", "SIZE", "STATUS", "CREATED")
	for i := range objects {
		o := &objects[i]
		table.Row(o.Key, humanBytes(o.SizeBytes), o.Status, o.CreatedAt)
	}
	if err := table.Render(); err != nil {
		return err
	}
	if len(objects) == 1 {
		term.Summary(w, "1 object")
	} else {
		term.Summary(w, fmt.Sprintf("%d objects", len(objects)))
	}
	return nil
}

// RmObject deletes one object matched by key, uid or filename.
func RmObject(ctx context.Context, rt *app.Runtime, ref string, yes bool) error {
	w := errOut(rt)
	bucketRef, key, err := splitObjectRef(ref)
	if err != nil {
		return err
	}
	workspace := strings.TrimSpace(rt.Config.Workspace)
	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	spin := term.StartLoading(w, "Finding object")
	objects, err := rt.API().ListBucketObjects(ctx, workspace, bucket.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findObject(objects, key, bucket.Name)
	if err != nil {
		return err
	}

	if !yes {
		if !rt.Streams.InputIsTerminal() {
			return fmt.Errorf("refusing to delete %s without confirmation; pass --yes to skip the prompt", target.Key)
		}
		confirmed, err := term.Confirm(rt.Streams.In, w,
			fmt.Sprintf("Delete %s from bucket %s? This cannot be undone", target.Key, bucket.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(w, term.Muted(w, "Cancelled"))
			return nil
		}
	}

	spin = term.StartLoading(w, "Deleting "+target.Key)
	if err := rt.API().DeleteBucketObject(ctx, workspace, target.UID); err != nil {
		spin.Stop()
		return err
	}
	spin.Stop()

	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, fmt.Sprintf("Deleted %s", target.Key)))
	return nil
}

func findObject(objects []api.AssetObject, ref, bucketName string) (api.AssetObject, error) {
	var matches []api.AssetObject
	for i := range objects {
		o := &objects[i]
		if o.UID == ref || o.Key == ref || o.Filename == ref {
			matches = append(matches, *o)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.AssetObject{}, fmt.Errorf("no object %q in bucket %s", ref, bucketName)
	default:
		keys := make([]string, 0, len(matches))
		for i := range matches {
			keys = append(keys, matches[i].Key)
		}
		return api.AssetObject{}, fmt.Errorf("multiple objects match %q: %s", ref, strings.Join(keys, ", "))
	}
}

// splitObjectRef splits a "<bucket>/<key>" reference at the first slash.
func splitObjectRef(ref string) (string, string, error) {
	bucket, key, found := strings.Cut(ref, "/")
	if !found || strings.TrimSpace(bucket) == "" || strings.TrimSpace(key) == "" {
		return "", "", fmt.Errorf("object reference must look like <bucket>/<key>, got %q", ref)
	}
	return bucket, key, nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func errOut(rt *app.Runtime) io.Writer {
	if rt.Streams.ErrOut != nil {
		return rt.Streams.ErrOut
	}
	return os.Stderr
}
