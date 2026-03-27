package vfs

import (
	"container/list"
	"sync"
)

// BlockCache implements an LRU Cache to keep uncompressed Zstd Deltas in RAM.
type BlockCache struct {
	capacity  int
	mu        sync.RWMutex
	items     map[uint32]*list.Element
	evictList *list.List
}

// cacheItem holds the key-value pair for the list element.
type cacheItem struct {
	blockID uint32
	data    []byte
}

// NewBlockCache creates a new LRU Cache for decoded blocks.
func NewBlockCache(capacity int) *BlockCache {
	return &BlockCache{
		capacity:  capacity,
		items:     make(map[uint32]*list.Element),
		evictList: list.New(),
	}
}

// Get fetches the decoded block pool if it is cached.
func (c *BlockCache) Get(blockID uint32) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[blockID]; ok {
		c.evictList.MoveToFront(ent)
		return ent.Value.(*cacheItem).data, true
	}
	return nil, false
}

// Put saves a decoded block pool to the cache.
func (c *BlockCache) Put(blockID uint32, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[blockID]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*cacheItem).data = data
		return
	}

	ent := c.evictList.PushFront(&cacheItem{blockID, data})
	c.items[blockID] = ent

	if c.evictList.Len() > c.capacity {
		c.removeOldest()
	}
}

func (c *BlockCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.evictList.Remove(ent)
		kv := ent.Value.(*cacheItem)
		delete(c.items, kv.blockID)
	}
}
