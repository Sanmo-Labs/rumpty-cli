package sync

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"golang.org/x/sync/errgroup"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

const transferWorkers = 8

type transferStats struct {
	Transferred int
	Deleted     int
	Bytes       int64
}

func executePlan(ctx context.Context, rt *app.Runtime, client *minio.Client, plan Plan, localRoot, s3Bucket, prefix string) (transferStats, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(transferWorkers)

	var (
		mu    sync.Mutex
		stats transferStats
	)
	spin := term.StartSpinner(rt.Streams.ErrOut, "Executing plan")
	defer spin.Stop()
	out := term.StopSpinnerOnWrite(rt.Streams.Out, spin)

	for _, action := range plan.Actions {
		g.Go(func() error {
			if err := runAction(ctx, client, action, localRoot, s3Bucket, prefix); err != nil {
				return fmt.Errorf("%s %s: %w", action.Kind, action.Key, err)
			}
			mu.Lock()
			defer mu.Unlock()
			switch action.Kind {
			case ActionDeleteRemote:
				stats.Deleted++
			default:
				stats.Transferred++
				stats.Bytes += action.Size
			}
			fmt.Fprintln(out, actionLine(action))
			return nil
		})
	}
	return stats, g.Wait()
}

func runAction(ctx context.Context, client *minio.Client, action Action, localRoot, s3Bucket, prefix string) error {
	remoteKey := action.Key
	if prefix != "" {
		remoteKey = prefix + "/" + action.Key
	}
	localPath := filepath.Join(localRoot, filepath.FromSlash(action.Key))

	switch action.Kind {
	case ActionUpload:
		_, err := client.FPutObject(ctx, s3Bucket, remoteKey, localPath, minio.PutObjectOptions{
			ContentType: contentTypeFor(action.Key),
		})
		return err
	case ActionDownload:
		return client.FGetObject(ctx, s3Bucket, remoteKey, localPath, minio.GetObjectOptions{})
	case ActionDeleteRemote:
		return client.RemoveObject(ctx, s3Bucket, remoteKey, minio.RemoveObjectOptions{})
	default:
		return fmt.Errorf("unknown action %q", action.Kind)
	}
}

func contentTypeFor(key string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(key))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func actionLine(action Action) string {
	switch action.Kind {
	case ActionUpload:
		return fmt.Sprintf("↑ %s (%s)", action.Key, humanBytes(action.Size))
	case ActionDownload:
		return fmt.Sprintf("↓ %s (%s)", action.Key, humanBytes(action.Size))
	case ActionDeleteRemote:
		return fmt.Sprintf("− %s (deleted)", action.Key)
	}
	return action.Key
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
