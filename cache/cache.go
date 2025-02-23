package cache

import (
	"sync"
	"time"
)

type Cache struct {
	data sync.Map
}

type cacheItem struct {
	value      []byte
	expiration int64
}

// NewCache initializes cache
func NewCache() *Cache {
	return &Cache{}
}

// Set stores a response in cache with TTL
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	expiration := time.Now().Add(ttl).Unix()
	c.data.Store(key, cacheItem{value, expiration})
}

// Get retrieves from cache
func (c *Cache) Get(key string) ([]byte, bool) {
	if item, found := c.data.Load(key); found {
		cacheItem := item.(cacheItem)
		if time.Now().Unix() < cacheItem.expiration {
			return cacheItem.value, true
		}
		c.data.Delete(key) // Remove expired cache
	}
	return nil, false
}
