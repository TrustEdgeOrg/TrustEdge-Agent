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
	listProcs  ProcessLister
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
		listProcs:  list,
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
		entry := e.identifyInstalled(app)
		entry.Installed = true
		byPath[pathKey(app.Path)] = &entry
	}

	procs, err := e.listProcs()
	if err != nil {
		e.logf("known-ai process list: %v", err)
		procs = nil
	}
	for _, proc := range procs {
		e.correlateProcess(byPath, proc)
	}

	out := make([]InventoryEntry, 0, len(byPath))
	for _, eptr := range byPath {
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
	ApplyPackageProvenance(&app)
	target := app.Path
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

func (e *Engine) correlateProcess(byPath map[string]*InventoryEntry, proc process.ProcessInfo) {
	exe := strings.TrimSpace(proc.Executable)
	if exe == "" {
		return
	}
	exe = posixPath(exe)
	appPath := EnclosingAppPath(exe)
	if appPath == "" {
		// Bare executable named like a candidate — identify without bundle path.
		id := identity.ApplicationIdentity{
			Executable:     posixBase(exe),
			ExecutablePath: exe,
			Path:           exe,
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
		return
	}

	key := pathKey(appPath)
	if existing, ok := byPath[key]; ok {
		existing.Running = true
		existing.Identification.Running = true
		existing.PIDs = appendPID(existing.PIDs, proc.PID)
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
	path := app.Path
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
