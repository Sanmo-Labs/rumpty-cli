package sync

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

const watchDebounce = 2 * time.Second

// Watch runs an initial sync, then keeps the bucket in sync as files change
// until ctx is cancelled. Sync failures are reported but do not stop the
// watcher; the next change retries.
func Watch(ctx context.Context, rt *app.Runtime, opts *Options) error {
	s, err := newSession(ctx, rt, opts)
	if err != nil {
		return err
	}
	if err := s.syncOnce(ctx, false); err != nil {
		return err
	}
	s.beat(nil)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if s.singleKey != "" {
		err = watcher.Add(s.localRoot)
	} else {
		err = addWatchesRecursive(watcher, s.localRoot)
	}
	if err != nil {
		return err
	}

	term.Statusf(rt.Streams.ErrOut, "Watching %s for changes; press Ctrl-C to stop", opts.LocalPath)

	debounce := time.NewTimer(watchDebounce)
	if !debounce.Stop() {
		<-debounce.C
	}

	for {
		select {
		case <-ctx.Done():
			term.Statusf(rt.Streams.ErrOut, "Stopped watching %s", opts.LocalPath)
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if s.singleKey != "" && event.Name != filepath.Join(s.localRoot, s.singleKey) {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addWatchesRecursive(watcher, event.Name)
				}
			}
			debounce.Reset(watchDebounce)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintln(rt.Streams.ErrOut, term.Error(rt.Streams.ErrOut, fmt.Sprintf("watch: %v", err)))
		case <-debounce.C:
			err := s.syncOnce(ctx, true)
			s.beat(err)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				fmt.Fprintln(rt.Streams.ErrOut, term.Error(rt.Streams.ErrOut, fmt.Sprintf("sync: %v", err)))
			}
		}
	}
}

func addWatchesRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		return watcher.Add(path)
	})
}
