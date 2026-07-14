//go:build darwin && cgo

package collect

/*
#cgo darwin LDFLAGS: -framework EndpointSecurity -lbsm
#cgo darwin CFLAGS: -x objective-c

#include <EndpointSecurity/EndpointSecurity.h>
#include <bsm/libbsm.h>
#include <stdlib.h>
#include <string.h>

extern void trusttwinEsDispatch(int eventType, int pid, int ppid, char *path, char *cmdline);

static char *es_join_exec_args(const es_event_exec_t *exec) {
	if (exec == NULL) {
		return NULL;
	}
	uint32_t n = es_exec_arg_count(exec);
	if (n == 0) {
		return NULL;
	}

	size_t total = 1;
	for (uint32_t i = 0; i < n; i++) {
		es_string_token_t arg = es_exec_arg(exec, i);
		if (arg.data == NULL || arg.length == 0) {
			continue;
		}
		total += (size_t)arg.length + 1;
	}

	char *out = (char *)malloc(total);
	if (out == NULL) {
		return NULL;
	}
	size_t pos = 0;
	for (uint32_t i = 0; i < n; i++) {
		es_string_token_t arg = es_exec_arg(exec, i);
		if (arg.data == NULL || arg.length == 0) {
			continue;
		}
		if (pos > 0) {
			out[pos++] = ' ';
		}
		memcpy(out + pos, arg.data, (size_t)arg.length);
		pos += (size_t)arg.length;
	}
	out[pos] = '\0';
	return out;
}

static void trusttwin_es_handler(es_client_t *client, const es_message_t *msg) {
	(void)client;
	if (msg == NULL) {
		return;
	}

	int pid = 0;
	int ppid = 0;
	char *path = NULL;
	char *cmdline = NULL;

	switch (msg->event_type) {
	case ES_EVENT_TYPE_NOTIFY_EXEC:
		if (msg->event.exec.target != NULL) {
			pid = audit_token_to_pid(msg->event.exec.target->audit_token);
			ppid = (int)msg->event.exec.target->ppid;
			if (msg->event.exec.target->executable != NULL &&
				msg->event.exec.target->executable->path.data != NULL) {
				path = strdup(msg->event.exec.target->executable->path.data);
			}
		}
		cmdline = es_join_exec_args(&msg->event.exec);
		trusttwinEsDispatch(1, pid, ppid, path, cmdline);
		break;
	case ES_EVENT_TYPE_NOTIFY_EXIT:
		if (msg->process != NULL) {
			pid = audit_token_to_pid(msg->process->audit_token);
			ppid = (int)msg->process->ppid;
			if (msg->process->executable != NULL &&
				msg->process->executable->path.data != NULL) {
				path = strdup(msg->process->executable->path.data);
			}
		}
		trusttwinEsDispatch(2, pid, ppid, path, NULL);
		break;
	default:
		break;
	}

	if (path != NULL) {
		free(path);
	}
	if (cmdline != NULL) {
		free(cmdline);
	}
}

static es_client_t *trusttwin_es_new_client(es_new_client_result_t *outResult) {
	es_client_t *client = NULL;
	es_new_client_result_t rc = es_new_client(&client, ^(es_client_t *c, const es_message_t *m) {
		trusttwin_es_handler(c, m);
	});
	if (outResult != NULL) {
		*outResult = rc;
	}
	return client;
}

static void trusttwin_es_delete_client(es_client_t *client) {
	if (client != NULL) {
		es_delete_client(client);
	}
}

static int trusttwin_es_subscribe_exec_exit(es_client_t *client) {
	es_event_type_t events[] = {
		ES_EVENT_TYPE_NOTIFY_EXEC,
		ES_EVENT_TYPE_NOTIFY_EXIT,
	};
	es_return_t rc = es_subscribe(client, events, sizeof(events) / sizeof(events[0]));
	return (int)rc;
}
*/
import "C"

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

type darwinProcessWatcher struct {
	log *log.Logger
	out chan ProcessChange
}

var (
	esMu       sync.Mutex
	esDispatch func(ProcessChange)
)

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	return &darwinProcessWatcher{log: logger}
}

func (w *darwinProcessWatcher) Run(ctx context.Context) <-chan ProcessChange {
	out := make(chan ProcessChange, 64)
	go func() {
		defer close(out)
		if err := w.run(ctx, out); err != nil && ctx.Err() == nil {
			w.logf("process watcher: %v (poll reconciliation continues)", err)
		}
	}()
	return out
}

func (w *darwinProcessWatcher) run(ctx context.Context, out chan<- ProcessChange) error {
	esMu.Lock()
	esDispatch = func(ch ProcessChange) {
		select {
		case out <- ch:
		default:
		}
	}
	esMu.Unlock()
	defer func() {
		esMu.Lock()
		esDispatch = nil
		esMu.Unlock()
	}()

	var result C.es_new_client_result_t
	client := C.trusttwin_es_new_client(&result)
	if client == nil {
		return esClientError(result)
	}
	defer C.trusttwin_es_delete_client(client)

	if rc := C.trusttwin_es_subscribe_exec_exit(client); rc != 0 {
		return esReturnError(int(rc))
	}

	w.logf("process watcher: Endpoint Security active")
	<-ctx.Done()
	return ctx.Err()
}

//export trusttwinEsDispatch
func trusttwinEsDispatch(eventType C.int, pid C.int, ppid C.int, path *C.char, cmdline *C.char) {
	esMu.Lock()
	dispatch := esDispatch
	esMu.Unlock()
	if dispatch == nil || pid <= 0 {
		return
	}

	exe := ""
	if path != nil {
		exe = C.GoString(path)
	}
	cmd := ""
	if cmdline != nil {
		cmd = truncateCmdline(C.GoString(cmdline))
	}
	comm := filepath.Base(exe)
	row := processRow{
		PID:        int(pid),
		PPID:       int(ppid),
		Comm:       comm,
		Executable: exe,
		Cmdline:    cmd,
	}

	var ch ProcessChange
	switch int(eventType) {
	case 1:
		ch = ProcessChange{Type: constants.TypeProcessStart, Payload: processPayload(row)}
	case 2:
		ch = ProcessChange{Type: constants.TypeProcessExit, Payload: processPayload(row)}
	default:
		return
	}

	dispatch(ch)
}

func esClientError(result C.es_new_client_result_t) error {
	switch result {
	case C.ES_NEW_CLIENT_RESULT_ERR_NOT_ENTITLED:
		return errNotEntitled
	case C.ES_NEW_CLIENT_RESULT_ERR_NOT_PERMITTED:
		return errNotPermitted
	default:
		return errESUnavailable
	}
}

func esReturnError(rc int) error {
	if rc == int(C.ES_RETURN_ERROR) {
		return errESUnavailable
	}
	return errESUnavailable
}

func (w *darwinProcessWatcher) logf(format string, args ...any) {
	if w.log != nil {
		w.log.Printf(format, args...)
	}
}
