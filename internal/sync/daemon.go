package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

type DaemonState struct {
	ID         string     `json:"id"`
	PID        int        `json:"pid"`
	LocalPath  string     `json:"local_path"`
	Bucket     string     `json:"bucket"`
	Prefix     string     `json:"prefix,omitempty"`
	Workspace  string     `json:"workspace"`
	StartedAt  time.Time  `json:"started_at"`
	LastSyncAt *time.Time `json:"last_sync_at,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
	LogFile    string     `json:"log_file"`
}

func daemonDir() (string, error) {
	dir, err := credentials.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sync-daemons"), nil
}

func daemonID(localRoot, bucket, prefix string) string {
	sum := sha256.Sum256([]byte(localRoot + "\x00" + bucket + "\x00" + prefix))
	return hex.EncodeToString(sum[:])[:12]
}

func statePath(id string) (string, error) {
	dir, err := daemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

func logPath(id string) (string, error) {
	dir, err := daemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".log"), nil
}

func writeState(state *DaemonState) error {
	path, err := statePath(state.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readState(id string) (*DaemonState, error) {
	path, err := statePath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func removeState(id string) {
	if path, err := statePath(id); err == nil {
		_ = os.Remove(path)
	}
}

func listStates() ([]*DaemonState, error) {
	dir, err := daemonDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var states []*DaemonState
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		state, err := readState(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].LocalPath < states[j].LocalPath })
	return states, nil
}

// SpawnDaemon starts a detached copy of the CLI running the watch loop and
// returns once the background process is up.
func SpawnDaemon(rt *app.Runtime, opts *Options) error {
	localRoot, err := filepath.Abs(opts.LocalPath)
	if err != nil {
		return err
	}
	id := daemonID(localRoot, opts.BucketName, opts.Prefix)
	if state, err := readState(id); err == nil && processAlive(state.PID) {
		return fmt.Errorf("a sync daemon for %s is already running (pid %d); run %q first",
			opts.LocalPath, state.PID, "rumpty sync stop "+opts.LocalPath)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logFilePath, err := logPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// context.Background is deliberate: the daemon child must outlive this
	// (short-lived) parent invocation rather than die with its context.
	cmd := exec.CommandContext(context.Background(), exe, daemonChildArgs(rt, localRoot, opts)...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// The token travels via environment, not argv, so it never shows up in ps.
	cmd.Env = append(os.Environ(), config.EnvToken+"="+rt.Config.Token)
	applyDetachAttrs(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	state := &DaemonState{
		ID:        id,
		PID:       pid,
		LocalPath: localRoot,
		Bucket:    opts.BucketName,
		Prefix:    opts.Prefix,
		Workspace: rt.Config.Workspace,
		StartedAt: time.Now().UTC(),
		LogFile:   logFilePath,
	}
	if err := writeState(state); err != nil {
		return err
	}

	term.Statusf(rt.Streams.ErrOut, "Sync daemon started (pid %d), watching %s", pid, opts.LocalPath)
	term.Statusf(rt.Streams.ErrOut, "Logs: %s", logFilePath)
	term.Statusf(rt.Streams.ErrOut, "Check with %q, stop with %q", "rumpty sync status", "rumpty sync stop "+opts.LocalPath)
	return nil
}

func daemonChildArgs(rt *app.Runtime, localRoot string, opts *Options) []string {
	bucketArg := opts.BucketName
	if opts.Prefix != "" {
		bucketArg += "/" + opts.Prefix
	}
	args := []string{
		"--ws", rt.Config.Workspace,
		"--api-url", rt.Config.APIURL,
		"sync", localRoot, bucketArg,
		"--daemon-run",
	}
	if opts.Delete {
		args = append(args, "--delete")
	}
	if opts.Public {
		args = append(args, "--public")
	}
	for _, pattern := range opts.Include {
		args = append(args, "--include", pattern)
	}
	for _, pattern := range opts.Exclude {
		args = append(args, "--exclude", pattern)
	}
	return args
}

// RunDaemon is the child side of --daemon: it supervises the watch loop,
// restarting it with backoff after crashes, and heartbeats into the state
// file so `rumpty sync status` reflects reality.
func RunDaemon(ctx context.Context, rt *app.Runtime, opts *Options) error {
	localRoot, err := filepath.Abs(opts.LocalPath)
	if err != nil {
		return err
	}
	id := daemonID(localRoot, opts.BucketName, opts.Prefix)
	logFilePath, _ := logPath(id)
	state := &DaemonState{
		ID:        id,
		PID:       os.Getpid(),
		LocalPath: localRoot,
		Bucket:    opts.BucketName,
		Prefix:    opts.Prefix,
		Workspace: rt.Config.Workspace,
		StartedAt: time.Now().UTC(),
		LogFile:   logFilePath,
	}
	if err := writeState(state); err != nil {
		return err
	}
	defer removeState(id)

	opts.Heartbeat = func(cycleErr error) {
		if cycleErr != nil {
			state.LastError = cycleErr.Error()
		} else {
			now := time.Now().UTC()
			state.LastError = ""
			state.LastSyncAt = &now
		}
		_ = writeState(state)
	}

	const minBackoff, maxBackoff = 5 * time.Second, time.Minute
	backoff := minBackoff
	for {
		started := time.Now()
		err := watchGuarded(ctx, rt, opts)
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			err = errors.New("watcher exited unexpectedly")
		}
		state.LastError = err.Error()
		_ = writeState(state)
		fmt.Fprintf(rt.Streams.ErrOut, "sync daemon: %v; restarting in %s\n", err, backoff)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if time.Since(started) > 5*time.Minute {
			backoff = minBackoff
		} else if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

func watchGuarded(ctx context.Context, rt *app.Runtime, opts *Options) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return Watch(ctx, rt, opts)
}

// StatusDaemons prints every known sync daemon and whether it is actually
// running.
func StatusDaemons(rt *app.Runtime) error {
	states, err := listStates()
	if err != nil {
		return err
	}
	if len(states) == 0 {
		fmt.Fprintln(rt.Streams.Out, "No sync daemons.")
		return nil
	}

	tw := tabwriter.NewWriter(rt.Streams.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PATH\tBUCKET\tSTATUS\tPID\tLAST SYNC\tLAST ERROR")
	for _, state := range states {
		status, pid := "stopped", "—"
		if processAlive(state.PID) {
			status = "running"
			pid = fmt.Sprintf("%d", state.PID)
		}
		target := state.Bucket
		if state.Prefix != "" {
			target += "/" + state.Prefix
		}
		lastErr := state.LastError
		if lastErr == "" {
			lastErr = "—"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			state.LocalPath, target, status, pid, humanAgo(state.LastSyncAt), lastErr)
	}
	return tw.Flush()
}

// StopDaemons terminates matching daemons. target matches the local path or
// bucket name; empty target requires either --all or exactly one daemon.
func StopDaemons(rt *app.Runtime, target string, all bool) error {
	states, err := listStates()
	if err != nil {
		return err
	}
	if len(states) == 0 {
		return errors.New("no sync daemons found")
	}

	var matched []*DaemonState
	switch {
	case all:
		matched = states
	case target == "":
		if len(states) > 1 {
			return errors.New("multiple sync daemons found; pass a path, a bucket, or --all")
		}
		matched = states
	default:
		abs, _ := filepath.Abs(target)
		for _, state := range states {
			if state.LocalPath == abs || strings.EqualFold(state.Bucket, target) {
				matched = append(matched, state)
			}
		}
		if len(matched) == 0 {
			return fmt.Errorf("no sync daemon matches %q", target)
		}
	}

	for _, state := range matched {
		if processAlive(state.PID) {
			if err := terminateProcess(state.PID); err != nil {
				return fmt.Errorf("stop daemon pid %d: %w", state.PID, err)
			}
			for range 50 {
				if !processAlive(state.PID) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if processAlive(state.PID) {
				_ = killProcess(state.PID)
			}
			term.Statusf(rt.Streams.ErrOut, "Stopped sync daemon for %s (pid %d)", state.LocalPath, state.PID)
		} else {
			term.Statusf(rt.Streams.ErrOut, "Cleaned up stale sync daemon entry for %s", state.LocalPath)
		}
		removeState(state.ID)
	}
	return nil
}

func humanAgo(t *time.Time) string {
	if t == nil {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
