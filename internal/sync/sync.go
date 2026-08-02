package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

type Options struct {
	LocalPath  string
	BucketName string
	Prefix     string
	Delete     bool
	DryRun     bool
	Restore    bool
	Public     bool
	Include    []string
	Exclude    []string
	Heartbeat  func(error)
}

type session struct {
	rt        *app.Runtime
	opts      Options
	workspace string
	localRoot string
	singleKey string
	filter    Filter
	target    string
	// remotePrefix is the effective object-key prefix applied to every remote
	// key: the bucket's server-provided KeyPrefix joined with any --prefix.
	remotePrefix string
	bucket       api.AssetBucket
	key          BucketKey
	keyCached    bool
	client       *minio.Client
	remoteCache  map[string]FileInfo
}

func Run(ctx context.Context, rt *app.Runtime, opts *Options) error {
	s, err := newSession(ctx, rt, opts)
	if err != nil {
		return err
	}
	return s.syncOnce(ctx, false)
}

func newSession(ctx context.Context, rt *app.Runtime, opts *Options) (*session, error) {
	s := &session{
		rt:        rt,
		opts:      *opts,
		workspace: strings.TrimSpace(rt.Config.Workspace),
		filter:    Filter{Include: opts.Include, Exclude: opts.Exclude},
	}

	localRoot, err := filepath.Abs(opts.LocalPath)
	if err != nil {
		return nil, err
	}
	if opts.Restore {
		if info, err := os.Stat(localRoot); err == nil && !info.IsDir() {
			return nil, fmt.Errorf("restore target %s is not a directory", opts.LocalPath)
		}
		if err := os.MkdirAll(localRoot, 0o750); err != nil {
			return nil, err
		}
	} else {
		info, err := os.Stat(localRoot)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			s.singleKey = filepath.Base(localRoot)
			localRoot = filepath.Dir(localRoot)
		}
	}
	s.localRoot = localRoot

	s.target = opts.BucketName
	if opts.Prefix != "" {
		s.target += "/" + opts.Prefix
	}

	s.bucket, err = ensureBucket(ctx, rt, s.workspace, opts.BucketName, !opts.Restore && !opts.DryRun, opts.Public)
	if err != nil {
		return nil, err
	}
	s.remotePrefix = joinPrefix(s.bucket.KeyPrefix, opts.Prefix)
	s.key, s.keyCached, err = ensureBucketKey(ctx, rt, s.workspace, &s.bucket)
	if err != nil {
		return nil, err
	}
	s.client, err = newS3Client(&s.key)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *session) syncOnce(ctx context.Context, quiet bool) error {
	remote, err := s.listRemoteState(ctx)
	if err != nil {
		return err
	}
	local, err := s.scanLocalState()
	if err != nil {
		return err
	}

	planLocal := s.filter.Apply(local)
	planRemote := s.filter.Apply(s.restrictToSingleKey(remote))

	var plan Plan
	if s.opts.Restore {
		plan = PlanDownload(planRemote, planLocal)
	} else {
		plan = PlanUpload(planLocal, planRemote, s.opts.Delete)
	}

	if len(plan.Actions) == 0 {
		s.remoteCache = remote
		if !quiet {
			fmt.Fprintf(s.rt.Streams.Out, "Already in sync with %s (%d files unchanged).\n", s.target, plan.Unchanged)
		}
		return nil
	}

	if s.opts.DryRun {
		for _, action := range plan.Actions {
			fmt.Fprintf(s.rt.Streams.Out, "would %s %s (%s)\n", action.Kind, action.Key, humanBytes(action.Size))
		}
		fmt.Fprintf(s.rt.Streams.Out, "Dry run: %s.\n", summarize(plan, transferStats{}, s.target, s.opts.Restore, true))
		return nil
	}

	stats, err := executePlan(ctx, s.rt, s.client, plan, s.localRoot, s.key.Bucket, s.remotePrefix)
	if err != nil {
		s.remoteCache = nil
		return err
	}
	applyActions(remote, plan.Actions)
	s.remoteCache = remote
	term.Statusf(s.rt.Streams.ErrOut, "%s", summarize(plan, stats, s.target, s.opts.Restore, false))
	return nil
}

func (s *session) listRemoteState(ctx context.Context) (map[string]FileInfo, error) {
	if s.remoteCache != nil {
		return s.remoteCache, nil
	}
	remote, err := listRemote(ctx, s.client, s.key.Bucket, s.remotePrefix)
	if err != nil && s.keyCached && isAuthError(err) {
		dropBucketKey(s.bucket.UID)
		s.keyCached = false
		if s.key, err = mintBucketKey(ctx, s.rt, s.workspace, &s.bucket); err != nil {
			return nil, err
		}
		if s.client, err = newS3Client(&s.key); err != nil {
			return nil, err
		}
		remote, err = listRemote(ctx, s.client, s.key.Bucket, s.remotePrefix)
	}
	if err != nil {
		return nil, fmt.Errorf("list bucket %s: %w", s.bucket.Name, err)
	}
	return remote, nil
}

func (s *session) scanLocalState() (map[string]FileInfo, error) {
	if s.singleKey != "" {
		path := filepath.Join(s.localRoot, s.singleKey)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return map[string]FileInfo{}, nil
			}
			return nil, err
		}
		return map[string]FileInfo{
			s.singleKey: {Key: s.singleKey, Size: info.Size(), ModTime: info.ModTime()},
		}, nil
	}
	return ScanLocal(s.localRoot)
}

// restrictToSingleKey narrows the remote view when syncing one file so
// --delete can never touch other objects. It copies rather than mutating the
// shared cache.
func (s *session) restrictToSingleKey(remote map[string]FileInfo) map[string]FileInfo {
	if s.singleKey == "" {
		return remote
	}
	out := map[string]FileInfo{}
	if info, ok := remote[s.singleKey]; ok {
		out[s.singleKey] = info
	}
	return out
}

func (s *session) beat(err error) {
	if s.opts.Heartbeat != nil {
		s.opts.Heartbeat(err)
	}
}

func applyActions(remote map[string]FileInfo, actions []Action) {
	now := time.Now()
	for _, action := range actions {
		switch action.Kind {
		case ActionUpload:
			remote[action.Key] = FileInfo{Key: action.Key, Size: action.Size, ModTime: now}
		case ActionDeleteRemote:
			delete(remote, action.Key)
		}
	}
}

// joinPrefix combines the bucket's server-provided key prefix with an optional
// user --prefix into a single slash-separated prefix with no leading/trailing
// slashes. Either part may be empty.
func joinPrefix(base, user string) string {
	parts := make([]string, 0, 2)
	for _, p := range []string{base, user} {
		if p = strings.Trim(strings.TrimSpace(p), "/"); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

func summarize(plan Plan, stats transferStats, target string, restore, dry bool) string {
	transfers, deletes, bytes := stats.Transferred, stats.Deleted, stats.Bytes
	if dry {
		for _, action := range plan.Actions {
			if action.Kind == ActionDeleteRemote {
				deletes++
			} else {
				transfers++
				bytes += action.Size
			}
		}
	}

	verb := "Synced"
	direction := "to"
	if restore {
		verb = "Restored"
		direction = "from"
	}
	s := fmt.Sprintf("%s %d files (%s) %s %s", verb, transfers, humanBytes(bytes), direction, target)
	if deletes > 0 {
		s += fmt.Sprintf(", %d remote files deleted", deletes)
	}
	if plan.Unchanged > 0 {
		s += fmt.Sprintf(", %d unchanged", plan.Unchanged)
	}
	return s
}
