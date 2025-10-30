package cache

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultTTL        = time.Minute
	DefaultMaxEntries = 1000
)

type CacheEntry struct {
	Info      os.FileInfo
	IsDir     bool
	Children  []os.FileInfo // For directory listings
	ExpiresAt time.Time
}

type NodeCache struct {
	entries    sync.Map // map[string]*CacheEntry
	ttl        time.Duration
	maxEntries int
}

func (c *NodeCache) NewEntry(f os.FileInfo) *CacheEntry {
	return &CacheEntry{
		Info:      f,
		IsDir:     f.IsDir(),
		Children:  nil,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func NewNodeCache(ttl time.Duration, maxEntries int) *NodeCache {
	return &NodeCache{
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

func (c *NodeCache) Get(path string) (*CacheEntry, bool) {
	if val, ok := c.entries.Load(path); ok {
		entry := val.(*CacheEntry)
		if time.Now().Before(entry.ExpiresAt) {
			return entry, true
		}
		c.entries.Delete(path)
	}
	return nil, false
}

func (c *NodeCache) Set(path string, entry *CacheEntry) {
	entry.ExpiresAt = time.Now().Add(c.ttl)
	c.entries.Store(path, entry)
}

func (c *NodeCache) Invalidate(path string) {
	c.entries.Delete(path)

	parent := filepath.Dir(path)
	if parent != path {
		c.entries.Delete(parent)
	}
}

func (c *NodeCache) InvalidateTree(path string) {
	c.entries.Range(func(key, value interface{}) bool {
		k := key.(string)
		if strings.HasPrefix(k, path) {
			c.entries.Delete(k)
		}
		return true
	})
}
