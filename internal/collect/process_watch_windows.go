//go:build windows

package collect

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/bi-zone/etw"
	"golang.org/x/sys/windows"
)

var kernelProcessProvider = windows.GUID{
	Data1: 0x22fb2cd6,
	Data2: 0x0e7b,
	Data3: 0x422b,
	Data4: [8]byte{0xa0, 0xc7, 0x2f, 0xad, 0x1f, 0xd0, 0xe7, 0x16},
}

type windowsProcessWatcher struct {
	log *log.Logger
}

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	return &windowsProcessWatcher{log: logger}
}

func (w *windowsProcessWatcher) Run(ctx context.Context) <-chan ProcessChange {
	out := make(chan ProcessChange, 64)
	go func() {
		defer close(out)
		if err := w.run(ctx, out); err != nil && ctx.Err() == nil {
			w.logf("process watcher: %v (poll reconciliation continues)", err)
		}
	}()
	return out
}

func (w *windowsProcessWatcher) run(ctx context.Context, out chan<- ProcessChange) error {
	session, err := openProcessETWSession()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := session.Process(func(e *etw.Event) {
			ch, ok := changeFromETW(e)
			if !ok {
				return
			}
			select {
			case out <- ch:
			case <-ctx.Done():
			}
		})
		if err != nil && ctx.Err() == nil {
			w.logf("process watcher: etw session ended: %v", err)
		}
	}()

	w.logf("process watcher: ETW kernel-process provider active (administrator recommended)")
	<-ctx.Done()
	if err := session.Close(); err != nil {
		w.logf("process watcher: close: %v", err)
	}
	wg.Wait()
	return ctx.Err()
}

func openProcessETWSession() (*etw.Session, error) {
	session, err := etw.NewSession(kernelProcessProvider)
	if err == nil {
		return session, nil
	}
	var exists etw.ExistsError
	if !errors.As(err, &exists) {
		return nil, err
	}
	if killErr := etw.KillSession(exists.SessionName); killErr != nil {
		return nil, killErr
	}
	return etw.NewSession(kernelProcessProvider)
}

func changeFromETW(e *etw.Event) (ProcessChange, bool) {
	if e == nil {
		return ProcessChange{}, false
	}
	switch e.Header.ID {
	case 1:
		return processStartFromETW(e)
	case 2:
		return processStopFromETW(e)
	default:
		return ProcessChange{}, false
	}
}

func processStartFromETW(e *etw.Event) (ProcessChange, bool) {
	props, err := e.EventProperties()
	if err != nil {
		return ProcessChange{}, false
	}
	pid := intProp(props, "ProcessID", "ProcessId")
	if pid <= 0 {
		return ProcessChange{}, false
	}
	ppid := intProp(props, "ParentProcessID", "ParentProcessId")
	image := stringProp(props, "ImageName", "ImageFileName", "CommandLine")
	comm := filepath.Base(image)
	row := processRow{PID: pid, PPID: ppid, Comm: comm, Executable: image}
	return ProcessChange{Type: constants.TypeProcessStart, Payload: processPayload(row)}, true
}

func processStopFromETW(e *etw.Event) (ProcessChange, bool) {
	props, err := e.EventProperties()
	if err != nil {
		return ProcessChange{}, false
	}
	pid := intProp(props, "ProcessID", "ProcessId")
	if pid <= 0 {
		return ProcessChange{}, false
	}
	ppid := intProp(props, "ParentProcessID", "ParentProcessId")
	image := stringProp(props, "ImageName", "ImageFileName")
	comm := filepath.Base(image)
	row := processRow{PID: pid, PPID: ppid, Comm: comm, Executable: image}
	return ProcessChange{
		Type: constants.TypeProcessExit,
		Payload: map[string]any{
			"pid":        row.PID,
			"ppid":       row.PPID,
			"user":       row.User,
			"comm":       row.Comm,
			"executable": row.Executable,
		},
	}, true
}

func intProp(props map[string]any, keys ...string) int {
	for _, key := range keys {
		v, ok := props[key]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case string:
			n = strings.TrimSpace(n)
			if i, err := strconv.Atoi(n); err == nil && i > 0 {
				return i
			}
		case int:
			return n
		case int32:
			return int(n)
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

func stringProp(props map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := props[key]
		if !ok {
			continue
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func (w *windowsProcessWatcher) logf(format string, args ...any) {
	if w.log != nil {
		w.log.Printf(format, args...)
	}
}
