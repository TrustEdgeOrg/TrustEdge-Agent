//go:build windows

package security

import (
	"context"
	"log"
	"time"

	"golang.org/x/sys/windows"
)

type windowsSecurityWatcher struct {
	log      *log.Logger
	debounce time.Duration
}

type regWatchTarget struct {
	hive windows.Handle
	path string
	name string
}

func newPlatformSecurityWatcher(logger *log.Logger) SecurityWatcher {
	return &windowsSecurityWatcher{log: logger, debounce: defaultSecurityWatchDebounce}
}

func (w *windowsSecurityWatcher) Run(ctx context.Context) <-chan struct{} {
	raw := make(chan struct{}, 8)
	go func() {
		defer close(raw)
		if err := w.run(ctx, raw); err != nil && ctx.Err() == nil {
			w.logf("security watcher: %v (poll reconciliation continues)", err)
		}
	}()
	return debounceWake(ctx, raw, w.debounce)
}

func (w *windowsSecurityWatcher) run(ctx context.Context, out chan<- struct{}) error {
	targets := []regWatchTarget{
		{windows.HKEY_CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "HKCU Run"},
		{windows.HKEY_CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\RunOnce`, "HKCU RunOnce"},
		{windows.HKEY_LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Run`, "HKLM Run"},
		{windows.HKEY_LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\RunOnce`, "HKLM RunOnce"},
		{windows.HKEY_LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, "HKLM WOW6432 Run"},
		{windows.HKEY_LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\RunOnce`, "HKLM WOW6432 RunOnce"},
		{windows.HKEY_LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services`, "HKLM Services"},
	}

	type watch struct {
		key   windows.Handle
		event windows.Handle
		name  string
	}
	var watches []watch
	defer func() {
		for _, wa := range watches {
			_ = windows.RegCloseKey(wa.key)
			_ = windows.CloseHandle(wa.event)
		}
	}()

	const access = windows.KEY_NOTIFY | windows.KEY_READ
	const filter = windows.REG_NOTIFY_CHANGE_NAME | windows.REG_NOTIFY_CHANGE_LAST_SET

	for _, t := range targets {
		var key windows.Handle
		err := windows.RegOpenKeyEx(t.hive, windows.StringToUTF16Ptr(t.path), 0, access, &key)
		if err != nil {
			w.logf("security watcher: skip %s: %v", t.name, err)
			continue
		}
		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			_ = windows.RegCloseKey(key)
			return err
		}
		subtree := uint32(0)
		if t.path == `SYSTEM\CurrentControlSet\Services` {
			subtree = 1
		}
		if err := windows.RegNotifyChangeKeyValue(key, subtree != 0, filter, event, true); err != nil {
			_ = windows.RegCloseKey(key)
			_ = windows.CloseHandle(event)
			w.logf("security watcher: notify %s: %v", t.name, err)
			continue
		}
		watches = append(watches, watch{key: key, event: event, name: t.name})
	}
	if len(watches) == 0 {
		return nil
	}

	w.logf("security watcher: registry notify active on %d keys", len(watches))

	handles := make([]windows.Handle, len(watches))
	for i, wa := range watches {
		handles[i] = wa.event
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Wait up to 1s so cancellation is noticed promptly.
		idx, err := waitForMultipleObjects(handles, 1000)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if idx < 0 {
			continue
		}
		wa := watches[idx]
		subtree := wa.name == "HKLM Services"
		if err := windows.RegNotifyChangeKeyValue(wa.key, subtree, filter, wa.event, true); err != nil {
			w.logf("security watcher: re-arm %s: %v", wa.name, err)
		}
		select {
		case out <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

// waitForMultipleObjects waits for any handle. Returns handle index, or -1 on timeout.
func waitForMultipleObjects(handles []windows.Handle, timeoutMs uint32) (int, error) {
	if len(handles) == 0 {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		return -1, nil
	}
	n, err := windows.WaitForMultipleObjects(handles, false, timeoutMs)
	switch {
	case err != nil:
		return -1, err
	case n == uint32(windows.WAIT_TIMEOUT):
		return -1, nil
	case n >= windows.WAIT_OBJECT_0 && n < windows.WAIT_OBJECT_0+uint32(len(handles)):
		return int(n - windows.WAIT_OBJECT_0), nil
	default:
		return -1, nil
	}
}

func (w *windowsSecurityWatcher) logf(format string, args ...any) {
	if w.log != nil {
		w.log.Printf(format, args...)
	}
}
