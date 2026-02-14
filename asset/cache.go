package asset

import "sync"

type Cache struct {
	mu      sync.Mutex
	entries map[string]any
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]any)}
}

func (c *Cache) Get(path string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[path]
	return v, ok
}

func (c *Cache) Put(path string, asset any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[path] = asset
}
