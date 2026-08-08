package apps

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// installIndex maps CLI command basenames to last-known installed identities.
// Used to resolve basename-only process names without consulting agent PATH.
type installIndex struct {
	mu   sync.RWMutex
	byName map[string][]identity.ApplicationIdentity // exact basename
	byPath map[string]identity.ApplicationIdentity   // pathKey -> install
}

func newInstallIndex() *installIndex {
	return &installIndex{
		byName: make(map[string][]identity.ApplicationIdentity),
		byPath: make(map[string]identity.ApplicationIdentity),
	}
}

func (idx *installIndex) rebuild(entries []InventoryEntry) {
	if idx == nil {
		return
	}
	byName := make(map[string][]identity.ApplicationIdentity)
	byPath := make(map[string]identity.ApplicationIdentity)
	for _, e := range entries {
		if !e.Installed {
			continue
		}
		id := e.Identity
		for _, name := range []string{id.Executable, posixBase(id.InvocationPath), posixBase(id.ResolvedPath)} {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			byName[name] = appendUniqueIdentity(byName[name], id)
		}
		for _, p := range []string{id.Path, id.ResolvedPath, id.InvocationPath, id.ExecutablePath} {
			p = posixPath(p)
			if p == "" {
				continue
			}
			byPath[pathKey(p)] = id
		}
	}
	idx.mu.Lock()
	idx.byName = byName
	idx.byPath = byPath
	idx.mu.Unlock()
}

func appendUniqueIdentity(list []identity.ApplicationIdentity, id identity.ApplicationIdentity) []identity.ApplicationIdentity {
	key := pathKey(id.ResolvedPath)
	if key == "" {
		key = pathKey(id.Path)
	}
	for _, existing := range list {
		ek := pathKey(existing.ResolvedPath)
		if ek == "" {
			ek = pathKey(existing.Path)
		}
		if ek == key {
			return list
		}
	}
	return append(list, id)
}

func (idx *installIndex) lookupByName(name string) []identity.ApplicationIdentity {
	if idx == nil || name == "" {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := idx.byName[name]
	if len(out) == 0 {
		return nil
	}
	cp := make([]identity.ApplicationIdentity, len(out))
	copy(cp, out)
	return cp
}

func (idx *installIndex) lookupByPath(path string) (identity.ApplicationIdentity, bool) {
	if idx == nil {
		return identity.ApplicationIdentity{}, false
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	id, ok := idx.byPath[pathKey(path)]
	return id, ok
}

// runtimeTracker tracks live ProcessKeys for known-AI processes.
type runtimeTracker struct {
	mu   sync.Mutex
	live map[process.ProcessKey]string // key -> product path key
}

func newRuntimeTracker() *runtimeTracker {
	return &runtimeTracker{live: make(map[process.ProcessKey]string)}
}

func (t *runtimeTracker) sync(active map[process.ProcessKey]string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.live = active
}

func (t *runtimeTracker) forgetPID(pid int) {
	if t == nil || pid <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for k := range t.live {
		if k.PID == pid {
			delete(t.live, k)
		}
	}
}

func (t *runtimeTracker) hasPID(pid int) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for k := range t.live {
		if k.PID == pid {
			return true
		}
	}
	return false
}

func basenameOnly(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return true
	}
	return !strings.Contains(path, string(filepath.Separator)) && !strings.Contains(path, "/")
}
