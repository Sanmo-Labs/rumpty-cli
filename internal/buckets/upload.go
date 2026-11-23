package buckets

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// Upload sends one local file to a bucket. The destination is
// "<bucket>/<key>"; the API appends a short unique suffix to the key, so
// the exact key cannot be requested. The final key is printed when the
// upload completes.
func Upload(ctx context.Context, rt *app.Runtime, localPath, dest string) error {
	w := errOut(rt)
	workspace := strings.TrimSpace(rt.Config.Workspace)

	bucketRef, key, err := splitObjectRef(dest)
	if err != nil {
		return err
	}
	bucket, err := resolveBucket(ctx, rt, bucketRef, workspace)
	if err != nil {
		return err
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory; rumpty bucket cp uploads one file at a time", localPath)
	}
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	filename, prefix := splitUploadKey(key, filepath.Base(localPath))
	contentType := uploadContentType(localPath, filename)
	if contentType == "" {
		return fmt.Errorf("cannot determine a supported content type for %s", localPath)
	}

	spin := term.StartLoading(w, "Reserving the upload")
	reserved, err := rt.API().CreateAssetUpload(ctx, workspace, bucket.UID, &api.CreateAssetUploadRequest{
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   info.Size(),
		KeyPrefix:   prefix,
	})
	spin.Stop()
	if err != nil {
		return err
	}

	spin = term.StartLoading(w, "Uploading "+filepath.Base(localPath))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reserved.UploadURL, file)
	if err != nil {
		return err
	}
	for k, v := range reserved.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: the upload URL is a presigned URL issued by the Rumpty API
	if err != nil {
		spin.Stop()
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		spin.Stop()
		return fmt.Errorf("upload failed: %s", strings.TrimSpace(string(body)))
	}
	spin.Stop()

	spin = term.StartLoading(w, "Completing the upload")
	object, err := rt.API().CompleteAssetUpload(ctx, workspace, reserved.ObjectID)
	spin.Stop()
	if err != nil {
		return err
	}

	out := rt.Streams.Out
	fmt.Fprintln(out, term.Success(out, fmt.Sprintf("Uploaded %s (%s) to %s", localPath, humanBytes(object.SizeBytes), object.Key)))
	return nil
}

// splitUploadKey turns the requested "<bucket>/<key>" into the API's
// filename and key_prefix: the last path segment stays the filename when it
// looks like one, everything before it becomes the prefix.
func splitUploadKey(key, fallback string) (string, string) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return fallback, ""
	}
	dir := path.Dir(key)
	base := path.Base(key)
	if dir == "." || dir == "/" {
		return base, ""
	}
	return base, dir
}

// uploadContentType picks a content type the API accepts for the file,
// guessing plain text only when the extension maps to a type it allows.
func uploadContentType(localPath, filename string) string {
	names := []string{filename, path.Base(localPath)}
	for _, name := range names {
		ct := mime.TypeByExtension(strings.ToLower(path.Ext(name)))
		base := ct
		if i := strings.IndexByte(base, ';'); i >= 0 {
			base = base[:i]
		}
		if uploadContentTypeAllowed(base) {
			return base
		}
	}
	return "text/plain"
}

// uploadContentTypeAllowed reports whether the API accepts a content type.
func uploadContentTypeAllowed(ct string) bool {
	switch strings.ToLower(ct) {
	case "application/pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.apple.mpegurl", "application/x-mpegurl", "video/mp2t",
		"image/gif", "image/jpeg", "image/png", "image/svg+xml", "image/webp",
		"text/csv", "text/html", "text/markdown", "text/x-markdown", "text/plain",
		"video/mp4", "video/quicktime", "video/webm", "video/x-matroska", "video/x-msvideo":
		return true
	}
	return false
}
