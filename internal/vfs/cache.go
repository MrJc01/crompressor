package vfs

import (
	"container/list"
	"sync"
)

// cacheItem holds the key-value pair for the list element.
type cacheItem struct {
	key   interface{} // uint32 (L1 Zstd Pool) or int64 (L2 Decompressed Chunk)
	data  []byte
	bytes int64
}

// MemoryCache implements a Byte-Aware LRU Cache (SRE Limits).
type MemoryCache struct {
	maxBytes   int64
	totalBytes int64
	mu         sync.RWMutex
	items      map[interface{}]*list.Element
	evictList  *list.List
}

// NewMemoryCache creates a new unified LRU Cache limited strictly by Megabytes.
func NewMemoryCache(maxMB int) *MemoryCache {
	if maxMB <= 0 {
		maxMB = 64 // SRE Fallback 64MB
	}
	return &MemoryCache{
		maxBytes:  int64(maxMB) * 1024 * 1024,
		items:     make(map[interface{}]*list.Element),
		evictList: list.New(),
	}
}

// Get fetches the data if it is cached.
func (c *MemoryCache) Get(key interface{}) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		return ent.Value.(*cacheItem).data, true
	}
	return nil, false
}

// Put saves data to the cache, evicting oldest until it fits within maxBytes.
func (c *MemoryCache) Put(key interface{}, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	size := int64(len(data))

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		oldSize := ent.Value.(*cacheItem).bytes
		c.totalBytes += size - oldSize
		ent.Value.(*cacheItem).data = data
		ent.Value.(*cacheItem).bytes = size
	} else {
		ent := c.evictList.PushFront(&cacheItem{key, data, size})
		c.items[key] = ent
		c.totalBytes += size
	}

	// Strictly limit memory by evicting items
	for c.totalBytes > c.maxBytes && c.evictList.Len() > 0 {
		c.removeOldest()
	}
}

func (c *MemoryCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.evictList.Remove(ent)
		kv := ent.Value.(*cacheItem)
		delete(c.items, kv.key)
		c.totalBytes -= kv.bytes
	}
}
