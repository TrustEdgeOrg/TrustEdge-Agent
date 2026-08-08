package identity

import (
	"container/list"
	"sync"
	"time"
)

// DefaultCacheCapacity is the maximum number of cached application identities.
const DefaultCacheCapacity = 256

// CacheKey uniquely identifies a cached application observation.
type CacheKey struct {
	Path string
	// ModTimeUnixNano and Size detect underlying file changes.
	ModTimeUnixNano int64
	Size            int64
}

// CacheEntry stores expensive identity results for an application path.
type CacheEntry struct {
	Key              CacheKey
	Identity         ApplicationIdentity
	Identification   IdentificationResult
	HasIdentity      bool
	HasIdentification bool
	StoredAt         time.Time
}

// Cache is a bounded LRU cache for identity extraction and matching results.
type Cache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[CacheKey]*list.Element
}

type cacheItem struct {
	key   CacheKey
	entry CacheEntry
}

// NewCache returns an LRU cache with the given capacity (minimum 1).
func NewCache(capacity int) *Cache {
	if capacity < 1 {
		capacity = DefaultCacheCapacity
	}
	return &Cache{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[CacheKey]*list.Element),
	}
}

// Get returns a cached entry for key.
func (c *Cache) Get(key CacheKey) (CacheEntry, bool) {
	if c == nil {
		return CacheEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return CacheEntry{}, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*cacheItem).entry, true
}

// PutIdentity stores or updates identity extraction for key.
func (c *Cache) PutIdentity(key CacheKey, id ApplicationIdentity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		item := el.Value.(*cacheItem)
		item.entry.Identity = id
		item.entry.HasIdentity = true
		item.entry.StoredAt = time.Now()
		c.ll.MoveToFront(el)
		return
	}
	entry := CacheEntry{
		Key:         key,
		Identity:    id,
		HasIdentity: true,
		StoredAt:    time.Now(),
	}
	c.addLocked(key, entry)
}

// PutIdentification stores matcher results for key.
func (c *Cache) PutIdentification(key CacheKey, res IdentificationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		item := el.Value.(*cacheItem)
		item.entry.Identification = res
		item.entry.HasIdentification = true
		item.entry.StoredAt = time.Now()
		c.ll.MoveToFront(el)
		return
	}
	entry := CacheEntry{
		Key:               key,
		Identification:    res,
		HasIdentification: true,
		StoredAt:          time.Now(),
	}
	c.addLocked(key, entry)
}

// InvalidatePath removes all entries whose Path equals path.
func (c *Cache) InvalidatePath(path string) {
	if c == nil || path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var remove []CacheKey
	for k := range c.items {
		if k.Path == path {
			remove = append(remove, k)
		}
	}
	for _, k := range remove {
		c.removeKeyLocked(k)
	}
}

// Len returns the number of cached entries.
func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Cap returns the maximum capacity.
func (c *Cache) Cap() int {
	if c == nil {
		return 0
	}
	return c.capacity
}

func (c *Cache) addLocked(key CacheKey, entry CacheEntry) {
	if len(c.items) >= c.capacity {
		c.evictOldestLocked()
	}
	item := &cacheItem{key: key, entry: entry}
	el := c.ll.PushFront(item)
	c.items[key] = el
}

func (c *Cache) evictOldestLocked() {
	el := c.ll.Back()
	if el == nil {
		return
	}
	item := el.Value.(*cacheItem)
	c.ll.Remove(el)
	delete(c.items, item.key)
}

func (c *Cache) removeKeyLocked(key CacheKey) {
	el, ok := c.items[key]
	if !ok {
		return
	}
	c.ll.Remove(el)
	delete(c.items, key)
}

// FileFingerprint builds a CacheKey from path and FileInfo-like attributes.
func FileFingerprint(path string, modTimeUnixNano, size int64) CacheKey {
	return CacheKey{
		Path:            path,
		ModTimeUnixNano: modTimeUnixNano,
		Size:            size,
	}
}
