package buckets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Cp downloads one object from a bucket to a local file. The source is
// "<bucket>/<key>"; in a private bucket the download goes through a
// presigned read URL.
func Cp(ctx context.Context, rt *app.Runtime, source, localPath string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	bucketRef, key, err := splitObjectRef(source)
	if err != nil {
		return err
	}
	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	spin := term.StartLoading(w, "Finding "+key+" in "+bucket.Name)
	objects, err := rt.API().ListBucketObjects(ctx, workspace, bucket.UID)
	spin.Stop()
	if err != nil {
		return err
	}
	target, err := findObject(objects, key, bucket.Name)
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Requesting a download URL")
	urlResult, err := rt.API().CreateObjectReadURL(ctx, workspace, target.UID)
	spin.Stop()
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Downloading "+target.Key)
	n, err := downloadFile(ctx, urlResult.ReadURL, localPath)
	spin.Stop()
	if err != nil {
		return err
	}

	line := fmt.Sprintf("Downloaded %s (%s) to %s", target.Key, humanBytes(n), localPath)
	if urlResult.IsSigned && urlResult.ExpiresAt != "" {
		line += fmt.Sprintf("; the link stays valid until %s", urlResult.ExpiresAt)
	}
	fmt.Fprintln(rt.Streams.Out, term.Success(rt.Streams.Out, line))
	return nil
}

func downloadFile(ctx context.Context, readURL, localPath string) (int64, error) {
	if strings.TrimSpace(readURL) == "" {
		return 0, fmt.Errorf("the API returned no download URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readURL, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req) //nolint:gosec // the URL comes from the Rumpty API's read-url response, not user input
	if err != nil {
		return 0, fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("the download URL returned %s", resp.Status)
	}
	if dir := filepath.Dir(localPath); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return 0, err
		}
	}
	out, err := os.Create(localPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	return io.Copy(out, resp.Body)
}
