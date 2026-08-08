package apps

import (
	"container/list"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const cliAuxCacheCapacity = 256

// cliAuxCache stores symlink resolution and package provenance keyed by
// resolved-path (or invocation-path) fingerprints. Bounded LRU; invalidates
// automatically when the fingerprint key changes.
type cliAuxCache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[identity.CacheKey]*list.Element
}

type cliAuxItem struct {
	key        identity.CacheKey
	resolved   ResolvedExecutable
	hasResolve bool
	identity   identity.ApplicationIdentity
	hasProv    bool
}

func newCLIAuxCache(capacity int) *cliAuxCache {
	if capacity < 1 {
		capacity = cliAuxCacheCapacity
	}
	return &cliAuxCache{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[identity.CacheKey]*list.Element),
	}
}

func (c *cliAuxCache) getResolve(key identity.CacheKey) (ResolvedExecutable, bool) {
	if c == nil {
		return ResolvedExecutable{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return ResolvedExecutable{}, false
	}
	item := el.Value.(*cliAuxItem)
	if !item.hasResolve {
		return ResolvedExecutable{}, false
	}
	c.ll.MoveToFront(el)
	return item.resolved, true
}

func (c *cliAuxCache) putResolve(key identity.CacheKey, resolved ResolvedExecutable) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		item := el.Value.(*cliAuxItem)
		item.resolved = resolved
		item.hasResolve = true
		c.ll.MoveToFront(el)
		return
	}
	c.addLocked(key, &cliAuxItem{key: key, resolved: resolved, hasResolve: true})
}

func (c *cliAuxCache) getProvenance(key identity.CacheKey) (identity.ApplicationIdentity, bool) {
	if c == nil {
		return identity.ApplicationIdentity{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return identity.ApplicationIdentity{}, false
	}
	item := el.Value.(*cliAuxItem)
	if !item.hasProv {
		return identity.ApplicationIdentity{}, false
	}
	c.ll.MoveToFront(el)
	return item.identity, true
}

func (c *cliAuxCache) putProvenance(key identity.CacheKey, id identity.ApplicationIdentity) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		item := el.Value.(*cliAuxItem)
		item.identity = id
		item.hasProv = true
		c.ll.MoveToFront(el)
		return
	}
	c.addLocked(key, &cliAuxItem{key: key, identity: id, hasProv: true})
}

func (c *cliAuxCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *cliAuxCache) addLocked(key identity.CacheKey, item *cliAuxItem) {
	if len(c.items) >= c.capacity {
		el := c.ll.Back()
		if el != nil {
			old := el.Value.(*cliAuxItem)
			c.ll.Remove(el)
			delete(c.items, old.key)
		}
	}
	el := c.ll.PushFront(item)
	c.items[key] = el
}

// fingerprintPath builds a cache key from an on-disk path.
func fingerprintPath(path string) identity.CacheKey {
	path = posixPath(path)
	if path == "" {
		return identity.CacheKey{}
	}
	fi, err := statPathFn(path)
	if err != nil {
		return identity.FileFingerprint(path, 0, 0)
	}
	return identity.FileFingerprint(path, fi.ModTime().UnixNano(), fi.Size())
}

// ResolveExecutableCached resolves with fingerprint-keyed caching.
func ResolveExecutableCached(cache *cliAuxCache, invocation string) (ResolvedExecutable, error) {
	key := fingerprintPath(invocation)
	if key.Path != "" {
		if hit, ok := cache.getResolve(key); ok {
			return hit, nil
		}
	}
	resolved, err := ResolveExecutable(invocation)
	if err != nil {
		return resolved, err
	}
	if key.Path != "" {
		cache.putResolve(key, resolved)
	}
	return resolved, nil
}

// ApplyPackageProvenanceCached fills provenance, caching by resolved-path fingerprint.
func ApplyPackageProvenanceCached(cache *cliAuxCache, id *identity.ApplicationIdentity) {
	if id == nil {
		return
	}
	resolved := id.ResolvedPath
	if resolved == "" {
		resolved = id.ExecutablePath
	}
	if resolved == "" {
		resolved = id.Path
	}
	key := fingerprintPath(resolved)
	if key.Path != "" {
		if hit, ok := cache.getProvenance(key); ok {
			id.PackageManager = hit.PackageManager
			id.PackageIdentifier = hit.PackageIdentifier
			id.PackageVersion = hit.PackageVersion
			if id.Version == "" {
				id.Version = hit.Version
			}
			if id.EntryPoint == "" {
				id.EntryPoint = hit.EntryPoint
			}
			return
		}
	}
	ApplyPackageProvenance(id)
	if key.Path != "" {
		cache.putProvenance(key, *id)
	}
}
