package cache

import (
	"strings"
	"sync"
	"time"
)

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

type Cache[T any] struct {
	mu    sync.RWMutex
	items map[string]entry[T]
	ttl   time.Duration
}

func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		items: make(map[string]entry[T]),
		ttl:   ttl,
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}

		var zero T

		return zero, false
	}

	return e.value, true
}

func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	c.items[key] = entry[T]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache[T]) DeleteByPrefix(prefix string) {
	c.mu.Lock()
	for k := range c.items {
		if strings.HasPrefix(k, prefix) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

func (c *Cache[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
