package apps

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// RuntimeEvent is a cheap process lifecycle signal from EndpointSecurity or
// process polling. Expensive identity work must not run on the ES callback path.
type RuntimeEvent struct {
	Kind              string // exec | fork | exit
	PID               int
	PPID              int
	Executable        string
	Comm              string
	StartTimeUnixNano int64
}

// RuntimeFeed asynchronously feeds process lifecycle events into known-AI
// inventory refreshes. The ES/notify path only enqueues; a worker decides
// whether a candidate may be involved and wakes inventory polling.
type RuntimeFeed struct {
	log     *log.Logger
	matcher *identity.Matcher
	engine  *Engine
	events  chan RuntimeEvent
	wakes   chan struct{}

	mu           sync.Mutex
	interesting  map[int]process.ProcessKey // PID → key while believed live
}

// NewRuntimeFeed constructs a bounded async feed.
func NewRuntimeFeed(logger *log.Logger, matcher *identity.Matcher) *RuntimeFeed {
	if matcher == nil {
		matcher = identity.NewMatcher(identity.DefaultCatalog())
	}
	return &RuntimeFeed{
		log:         logger,
		matcher:     matcher,
		events:      make(chan RuntimeEvent, 256),
		wakes:       make(chan struct{}, 1),
		interesting: make(map[int]process.ProcessKey),
	}
}

// SetEngine optionally binds an Engine for EXIT bookkeeping / install lookups.
func (f *RuntimeFeed) SetEngine(engine *Engine) {
	if f == nil {
		return
	}
	f.engine = engine
}

// Wakes returns a channel signaled when inventory should be refreshed soon.
func (f *RuntimeFeed) Wakes() <-chan struct{} {
	return f.wakes
}

// ObserveChange translates a process collector change into a RuntimeEvent.
// Safe to call from ES dispatch / watcher goroutines: never blocks on identity work.
func (f *RuntimeFeed) ObserveChange(c collect.Change) {
	if f == nil {
		return
	}
	kind := ""
	switch c.Type {
	case constants.TypeProcessStart:
		kind = "exec"
	case constants.TypeProcessFork:
		kind = "fork"
	case constants.TypeProcessExit:
		kind = "exit"
	default:
		return
	}
	ev := RuntimeEvent{
		Kind:              kind,
		PID:               intFromAny(c.Payload["pid"]),
		PPID:              intFromAny(c.Payload["ppid"]),
		Executable:        stringFromPayload(c.Payload["executable"]),
		Comm:              stringFromPayload(c.Payload["comm"]),
		StartTimeUnixNano: int64FromAny(c.Payload["start_time_unix_nano"]),
	}
	select {
	case f.events <- ev:
	default:
		// Drop under pressure; poll reconciliation remains authoritative.
		f.logf("known-ai runtime feed: event queue full, dropping")
	}
}

// Run processes queued lifecycle events until ctx is done.
func (f *RuntimeFeed) Run(ctx context.Context) {
	if f == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-f.events:
			if f.shouldWake(ev) {
				f.signalWake()
			}
		}
	}
}

func (f *RuntimeFeed) shouldWake(ev RuntimeEvent) bool {
	if ev.Kind == "exit" {
		f.mu.Lock()
		_, tracked := f.interesting[ev.PID]
		delete(f.interesting, ev.PID)
		f.mu.Unlock()
		if f.engine != nil {
			f.engine.NoteExit(ev.PID)
		}
		// Wake when we tracked this PID, or exit path still looks like a candidate.
		return tracked || f.looksLikeCandidate(ev)
	}
	if !f.looksLikeCandidate(ev) {
		return false
	}
	f.mu.Lock()
	f.interesting[ev.PID] = process.ProcessKey{
		PID:               ev.PID,
		StartTimeUnixNano: ev.StartTimeUnixNano,
	}
	f.mu.Unlock()
	return true
}

func (f *RuntimeFeed) looksLikeCandidate(ev RuntimeEvent) bool {
	exe := ev.Executable
	comm := ev.Comm
	base := firstNonEmptyStr(comm, filepath.Base(exe))
	id := identity.ApplicationIdentity{
		Executable:     base,
		ExecutablePath: exe,
		Path:           EnclosingAppPath(exe),
	}
	if id.Path == "" {
		id.Path = exe
	}
	if basenameOnly(exe) || basenameOnly(base) {
		id.Executable = base
		id.Path = base
		id.ExecutablePath = base
	}
	// Candidate generation only — do not treat this as verification.
	res := f.matcher.Identify(id)
	return res.Product != nil
}

func (f *RuntimeFeed) signalWake() {
	select {
	case f.wakes <- struct{}{}:
	default:
	}
}

func (f *RuntimeFeed) logf(format string, args ...any) {
	if f.log != nil {
		f.log.Printf(format, args...)
	}
}

func stringFromPayload(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
