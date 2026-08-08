package apps

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const extensionMetaMaxBytes = 256 << 10 // 256 KiB package.json cap
const extensionMetaCacheCap = 256

// ExtensionPackageMeta is normalized package.json identity for an extension.
type ExtensionPackageMeta struct {
	Name        string
	Publisher   string
	Version     string
	DisplayName string
	// ExtensionID is publisher.name when both are present.
	ExtensionID string
}

type extensionMetaCacheEntry struct {
	fp   identity.CacheKey
	meta ExtensionPackageMeta
	ok   bool
}

type extensionMetaCache struct {
	mu    sync.Mutex
	order []string
	items map[string]extensionMetaCacheEntry
	cap   int
}

func newExtensionMetaCache(capacity int) *extensionMetaCache {
	if capacity <= 0 {
		capacity = extensionMetaCacheCap
	}
	return &extensionMetaCache{
		items: make(map[string]extensionMetaCacheEntry, capacity),
		cap:   capacity,
	}
}

func (c *extensionMetaCache) get(path string, fp identity.CacheKey) (ExtensionPackageMeta, bool, bool) {
	if c == nil {
		return ExtensionPackageMeta{}, false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[path]
	if !ok || e.fp != fp {
		return ExtensionPackageMeta{}, false, false
	}
	return e.meta, e.ok, true
}

func (c *extensionMetaCache) put(path string, fp identity.CacheKey, meta ExtensionPackageMeta, ok bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[path]; !exists {
		c.order = append(c.order, path)
		for len(c.order) > c.cap {
			old := c.order[0]
			c.order = c.order[1:]
			delete(c.items, old)
		}
	}
	c.items[path] = extensionMetaCacheEntry{fp: fp, meta: meta, ok: ok}
}

// ReadExtensionPackageJSON reads and normalizes package.json under installPath.
func ReadExtensionPackageJSON(installPath string) (ExtensionPackageMeta, error) {
	pkgPath := filepath.Join(installPath, "package.json")
	f, err := os.Open(pkgPath)
	if err != nil {
		return ExtensionPackageMeta{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, extensionMetaMaxBytes))
	if err != nil {
		return ExtensionPackageMeta{}, err
	}
	return parseExtensionPackageJSON(raw)
}

func parseExtensionPackageJSON(raw []byte) (ExtensionPackageMeta, error) {
	var doc struct {
		Name        string `json:"name"`
		Publisher   string `json:"publisher"`
		Version     string `json:"version"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ExtensionPackageMeta{}, err
	}
	name := strings.TrimSpace(doc.Name)
	publisher := strings.TrimSpace(doc.Publisher)
	meta := ExtensionPackageMeta{
		Name:        name,
		Publisher:   publisher,
		Version:     strings.TrimSpace(doc.Version),
		DisplayName: strings.TrimSpace(doc.DisplayName),
	}
	if publisher != "" && name != "" {
		meta.ExtensionID = publisher + "." + name
	}
	return meta, nil
}

func applyExtensionPackageMeta(app identity.ApplicationIdentity, meta ExtensionPackageMeta) identity.ApplicationIdentity {
	if meta.ExtensionID != "" {
		app.PackageIdentifier = meta.ExtensionID
	}
	if meta.Version != "" {
		app.Version = meta.Version
		app.PackageVersion = meta.Version
	}
	if meta.Name != "" {
		app.Executable = meta.Name
		app.EntryPoint = meta.Name
	}
	return app
}

func init() {
	metaCache := newExtensionMetaCache(extensionMetaCacheCap)
	enrichExtensionMetadata = func(e *Engine, app identity.ApplicationIdentity, installPath string) identity.ApplicationIdentity {
		pkgPath := filepath.Join(installPath, "package.json")
		fi, err := os.Stat(pkgPath)
		if err != nil {
			return app
		}
		fp := identity.CacheKey{Path: pkgPath, ModTimeUnixNano: fi.ModTime().UnixNano(), Size: fi.Size()}
		if meta, okParse, hit := metaCache.get(pkgPath, fp); hit {
			if !okParse {
				return app
			}
			return applyExtensionPackageMeta(app, meta)
		}
		meta, err := ReadExtensionPackageJSON(installPath)
		if err != nil || meta.ExtensionID == "" {
			metaCache.put(pkgPath, fp, ExtensionPackageMeta{}, false)
			return app
		}
		metaCache.put(pkgPath, fp, meta, true)
		return applyExtensionPackageMeta(app, meta)
	}
}
