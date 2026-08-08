package apps

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// ProcessLister returns currently running processes.
type ProcessLister func() ([]process.ProcessInfo, error)

// Engine correlates installed applications and running processes through the
// shared identity matcher. It does not duplicate identification logic.
type Engine struct {
	log        *log.Logger
	discoverer Discoverer
	signer     Signer
	matcher    *identity.Matcher
	cache      *identity.Cache
	cliCache   *cliAuxCache
	listProcs  ProcessLister
	installs   *installIndex
	runtime    *runtimeTracker
}

// EngineConfig configures a correlation engine.
type EngineConfig struct {
	Logger     *log.Logger
	Discoverer Discoverer
	Signer     Signer
	Matcher    *identity.Matcher
	Cache      *identity.Cache
	ListProcs  ProcessLister
}

// NewEngine constructs a correlation engine with platform defaults.
func NewEngine(cfg EngineConfig) *Engine {
	d := cfg.Discoverer
	if d == nil {
		d = NewDiscoverer(cfg.Logger)
	}
	s := cfg.Signer
	if s == nil {
		s = NewSigner()
	}
	m := cfg.Matcher
	if m == nil {
		m = identity.NewMatcher(identity.DefaultCatalog())
	}
	c := cfg.Cache
	if c == nil {
		c = identity.NewCache(identity.DefaultCacheCapacity)
	}
	list := cfg.ListProcs
	if list == nil {
		list = process.Snapshot
	}
	return &Engine{
		log:        cfg.Logger,
		discoverer: d,
		signer:     s,
		matcher:    m,
		cache:      c,
		cliCache:   newCLIAuxCache(cliAuxCacheCapacity),
		listProcs:  list,
		installs:   newInstallIndex(),
		runtime:    newRuntimeTracker(),
	}
}

// InventoryEntry is one known-AI product installation observation.
type InventoryEntry struct {
	Identity         identity.ApplicationIdentity
	Identification   identity.IdentificationResult
	Installed        bool
	Running          bool
	PIDs             []int
}

// Inventory builds the current known-AI software inventory.
func (e *Engine) Inventory() ([]InventoryEntry, error) {
	installed, err := e.discoverer.Discover()
	if err != nil {
		e.logf("known-ai discover: %v", err)
		installed = nil
	}

	byPath := make(map[string]*InventoryEntry)
	for _, app := range installed {
		app.Path = posixPath(app.Path)
		if app.ExecutablePath != "" {
			app.ExecutablePath = posixPath(app.ExecutablePath)
		}
		if app.ResolvedPath != "" {
			app.ResolvedPath = posixPath(app.ResolvedPath)
		}
		if app.InvocationPath != "" {
			app.InvocationPath = posixPath(app.InvocationPath)
		}
		eptr := new(InventoryEntry)
		*eptr = e.identifyInstalled(app)
		eptr.Installed = true
		byPath[pathKey(app.Path)] = eptr
		// Also index by resolved/invocation so runtime path matches land.
		if app.ResolvedPath != "" {
			byPath[pathKey(app.ResolvedPath)] = eptr
		}
		if app.InvocationPath != "" {
			byPath[pathKey(app.InvocationPath)] = eptr
		}
	}

	// Index installs before process correlation so basename-only EXEC can map
	// via last-known discovery paths (not agent PATH).
	{
		pre := make([]InventoryEntry, 0, len(byPath))
		seen := make(map[*InventoryEntry]struct{})
		for _, eptr := range byPath {
			if _, ok := seen[eptr]; ok {
				continue
			}
			seen[eptr] = struct{}{}
			pre = append(pre, *eptr)
		}
		e.installs.rebuild(pre)
	}

	procs, err := e.listProcs()
	if err != nil {
		e.logf("known-ai process list: %v", err)
		procs = nil
	}
	activeKeys := make(map[process.ProcessKey]string)
	for _, proc := range procs {
		e.correlateProcess(byPath, proc, activeKeys)
	}
	e.runtime.sync(activeKeys)

	out := make([]InventoryEntry, 0, len(byPath))
	seenPtr := make(map[*InventoryEntry]struct{})
	for _, eptr := range byPath {
		if _, ok := seenPtr[eptr]; ok {
			continue
		}
		seenPtr[eptr] = struct{}{}
		// Drop entries that never matched a known product.
		if eptr.Identification.Product == nil {
			continue
		}
		sort.Ints(eptr.PIDs)
		out = append(out, *eptr)
	}
	sort.Slice(out, func(i, j int) bool {
		pi, pj := out[i].Identification.Product, out[j].Identification.Product
		if pi.ID != pj.ID {
			return pi.ID < pj.ID
		}
		return out[i].Identity.Path < out[j].Identity.Path
	})
	e.installs.rebuild(out)
	return out, nil
}

func (e *Engine) identifyInstalled(app identity.ApplicationIdentity) InventoryEntry {
	key := cacheKeyForApp(app)
	if key.Path != "" {
		if cached, ok := e.cache.Get(key); ok && cached.HasIdentity && cached.HasIdentification {
			res := cached.Identification
			res.Installed = true
			return InventoryEntry{
				Identity:       cached.Identity,
				Identification: res,
				Installed:      true,
			}
		}
	}

	id := e.enrich(app)
	res := e.matcher.Identify(id)
	res.Installed = true
	if key.Path != "" {
		e.cache.PutIdentity(key, id)
		e.cache.PutIdentification(key, res)
	}
	return InventoryEntry{
		Identity:       id,
		Identification: res,
		Installed:      true,
	}
}

func (e *Engine) enrich(app identity.ApplicationIdentity) identity.ApplicationIdentity {
	ApplyPackageProvenanceCached(e.cliCache, &app)
	target := app.ResolvedPath
	if target == "" {
		target = app.Path
	}
	if target == "" {
		target = app.ExecutablePath
	}
	if target == "" || e.signer == nil {
		return app
	}
	info, err := ExtractAndValidate(e.signer, target)
	if err != nil {
		e.logf("known-ai signing %s: %v", filepath.Base(target), err)
		// Still record checked=false; partial extract may have fields.
		ApplySigning(&app, info)
		return app
	}
	ApplySigning(&app, info)
	return app
}

func (e *Engine) correlateProcess(byPath map[string]*InventoryEntry, proc process.ProcessInfo, activeKeys map[process.ProcessKey]string) {
	exe := strings.TrimSpace(proc.Executable)
	if exe == "" {
		exe = strings.TrimSpace(proc.Comm)
	}
	if exe == "" {
		return
	}
	exe = e.resolveRuntimeExecutable(exe, proc.Comm)
	if exe == "" {
		return
	}
	exe = posixPath(exe)
	appPath := EnclosingAppPath(exe)
	if appPath == "" {
		// CLI / bare executable path.
		if e.correlateInstalledCLI(byPath, proc, exe, activeKeys) {
			return
		}
		id := identity.ApplicationIdentity{
			Executable:     firstNonEmpty(posixBase(exe), proc.Comm),
			ExecutablePath: exe,
			Path:           exe,
			ResolvedPath:   exe,
		}
		res := e.matcher.Identify(id)
		if res.Product == nil {
			return
		}
		// Name-only running process: do not invent an install path.
		key := "running:" + pathKey(exe)
		if existing, ok := byPath[key]; ok {
			existing.Running = true
			existing.PIDs = appendPID(existing.PIDs, proc.PID)
			if activeKeys != nil {
				activeKeys[proc.Key()] = key
			}
			return
		}
		res.Running = true
		res.Installed = false
		byPath[key] = &InventoryEntry{
			Identity:       id,
			Identification: res,
			Installed:      false,
			Running:        true,
			PIDs:           []int{proc.PID},
		}
		if activeKeys != nil {
			activeKeys[proc.Key()] = key
		}
		return
	}

	key := pathKey(appPath)
	if existing, ok := byPath[key]; ok {
		existing.Running = true
		existing.Identification.Running = true
		existing.PIDs = appendPID(existing.PIDs, proc.PID)
		if activeKeys != nil {
			activeKeys[proc.Key()] = key
		}
		return
	}

	// Running from a path not seen in /Applications scan (e.g. custom location).
	app, ok := ResolveBundle(appPath)
	if !ok {
		app = identity.ApplicationIdentity{
			Path:           appPath,
			Executable:     posixBase(exe),
			ExecutablePath: exe,
		}
	} else {
		app.Path = posixPath(app.Path)
		if app.ExecutablePath != "" {
			app.ExecutablePath = posixPath(app.ExecutablePath)
		}
	}
	entry := e.identifyInstalled(app)
	entry.Installed = true // on disk at appPath
	entry.Running = true
	entry.Identification.Running = true
	entry.Identification.Installed = true
	entry.PIDs = []int{proc.PID}
	byPath[key] = &entry
	if activeKeys != nil {
		activeKeys[proc.Key()] = key
	}
}

func (e *Engine) correlateInstalledCLI(byPath map[string]*InventoryEntry, proc process.ProcessInfo, exe string, activeKeys map[process.ProcessKey]string) bool {
	candidates := []string{exe}
	if id, ok := e.installs.lookupByPath(exe); ok {
		candidates = append(candidates, id.Path, id.ResolvedPath, id.InvocationPath)
	}
	for _, cand := range candidates {
		cand = posixPath(cand)
		if cand == "" {
			continue
		}
		if existing, ok := byPath[pathKey(cand)]; ok && existing.Installed {
			existing.Running = true
			existing.Identification.Running = true
			existing.PIDs = appendPID(existing.PIDs, proc.PID)
			if activeKeys != nil {
				activeKeys[proc.Key()] = pathKey(cand)
			}
			return true
		}
	}
	return false
}

// resolveRuntimeExecutable maps a process executable to a filesystem path
// using discovery cache / install index — never the agent process PATH.
func (e *Engine) resolveRuntimeExecutable(exe, comm string) string {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		exe = strings.TrimSpace(comm)
	}
	if exe == "" {
		return ""
	}
	if !basenameOnly(exe) {
		if r, err := ResolveExecutableCached(e.cliCache, exe); err == nil && r.ResolvedPath != "" {
			return r.ResolvedPath
		}
		return posixPath(exe)
	}
	name := exe
	if installs := e.installs.lookupByName(name); len(installs) > 0 {
		id := installs[0]
		if id.ResolvedPath != "" {
			return id.ResolvedPath
		}
		if id.InvocationPath != "" {
			return id.InvocationPath
		}
		return id.Path
	}
	if installs := e.installs.lookupByName(comm); len(installs) > 0 {
		id := installs[0]
		if id.ResolvedPath != "" {
			return id.ResolvedPath
		}
		return id.Path
	}
	// Basename with no known install: keep basename for candidate matching only.
	return name
}

// NoteExit clears runtime tracking for a PID (called from RuntimeFeed on EXIT).
func (e *Engine) NoteExit(pid int) {
	if e == nil {
		return
	}
	e.runtime.forgetPID(pid)
}

func appendPID(pids []int, pid int) []int {
	if pid <= 0 {
		return pids
	}
	for _, p := range pids {
		if p == pid {
			return pids
		}
	}
	return append(pids, pid)
}

func cacheKeyForApp(app identity.ApplicationIdentity) identity.CacheKey {
	// Prefer resolved executable path for CLIs so symlink retargets invalidate.
	path := app.ResolvedPath
	if path == "" {
		path = app.Path
	}
	if path == "" {
		path = app.ExecutablePath
	}
	if path == "" {
		return identity.CacheKey{}
	}
	fiPath := path
	if strings.HasSuffix(strings.ToLower(path), ".app") {
		fiPath = filepath.Join(path, "Contents", "Info.plist")
	}
	fi, err := os.Stat(fiPath)
	if err != nil {
		fi, err = os.Stat(path)
		if err != nil {
			return identity.FileFingerprint(path, 0, 0)
		}
	}
	return identity.FileFingerprint(path, fi.ModTime().UnixNano(), fi.Size())
}

func (e *Engine) logf(format string, args ...any) {
	if e.log != nil {
		e.log.Printf(format, args...)
	}
}
