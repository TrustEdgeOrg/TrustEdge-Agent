//go:build darwin

package security

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

type darwinSecurityWatcher struct {
	log      *log.Logger
	debounce time.Duration
	dirs     []string // optional override for tests
}

func newPlatformSecurityWatcher(logger *log.Logger) SecurityWatcher {
	return &darwinSecurityWatcher{log: logger, debounce: defaultSecurityWatchDebounce}
}

func (w *darwinSecurityWatcher) Run(ctx context.Context) <-chan struct{} {
	raw := make(chan struct{}, 8)
	go func() {
		defer close(raw)
		dirs := w.dirs
		if len(dirs) == 0 {
			dirs = securityWatchDirs()
		}
		if err := w.run(ctx, dirs, raw); err != nil && ctx.Err() == nil {
			w.logf("security watcher: %v (poll reconciliation continues)", err)
		}
	}()
	return debounceWake(ctx, raw, w.debounce)
}

func (w *darwinSecurityWatcher) run(ctx context.Context, dirs []string, out chan<- struct{}) error {
	if len(dirs) == 0 {
		return nil
	}

	kq, err := unix.Kqueue()
	if err != nil {
		return err
	}
	defer unix.Close(kq)

	type watched struct {
		fd  int
		dir string
	}
	var watches []watched
	defer func() {
		for _, wa := range watches {
			_ = unix.Close(wa.fd)
		}
	}()

	events := make([]unix.Kevent_t, 0, len(dirs))
	for _, dir := range dirs {
		fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			if !os.IsNotExist(err) {
				w.logf("security watcher: skip %s: %v", dir, err)
			}
			continue
		}
		watches = append(watches, watched{fd: fd, dir: dir})
		events = append(events, unix.Kevent_t{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_VNODE,
			Flags:  unix.EV_ADD | unix.EV_CLEAR,
			Fflags: unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_RENAME | unix.NOTE_EXTEND | unix.NOTE_ATTRIB,
		})
	}
	if len(events) == 0 {
		return nil
	}
	if _, err := unix.Kevent(kq, events, nil, nil); err != nil {
		return err
	}

	w.logf("security watcher: kqueue active on %d launch paths", len(watches))

	buf := make([]unix.Kevent_t, len(watches))
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Short timeout so we can notice cancellation without a dedicated wake fd.
		timeout := &unix.Timespec{Sec: 1}
		n, err := unix.Kevent(kq, nil, buf, timeout)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if n <= 0 {
			continue
		}
		select {
		case out <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Already have a pending wake; debounceWake coalesces.
		}
	}
}

func securityWatchDirs() []string {
	home, _ := homeDirFn()
	dirs := []string{
		"/Library/LaunchAgents",
		"/Library/LaunchDaemons",
	}
	if home != "" {
		dirs = append([]string{filepath.Join(home, "Library", "LaunchAgents")}, dirs...)
	}
	return dirs
}

func (w *darwinSecurityWatcher) logf(format string, args ...any) {
	if w.log != nil {
		w.log.Printf(format, args...)
	}
}
