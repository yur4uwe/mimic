package cache

import (
	"fmt"
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
	Children  []os.FileInfo
	ExpiresAt time.Time
}

type NodeCache struct {
	entries    sync.Map // map[string]*CacheEntry
	ttl        time.Duration
	maxEntries int
}

// String implements fmt.Stringer and returns a concise summary.
func (c *NodeCache) String() string {
	return c.Summary(10)
}

// Summary returns a concise human-readable representation of the cache.
// If maxKeys > 0 it includes up to that many entry keys in the output;
// if maxKeys <= 0 it omits the keys and only prints counts/settings.
func (c *NodeCache) Summary(maxKeys int) string {
	total := 0
	keys := make([]string, 0, 8)

	c.entries.Range(func(k, v interface{}) bool {
		total++
		if maxKeys > 0 && len(keys) < maxKeys {
			if ks, ok := k.(string); ok {
				keys = append(keys, ks)
			} else {
				keys = append(keys, fmt.Sprintf("%v", k))
			}
		}
		return true
	})

	if maxKeys > 0 {
		return fmt.Sprintf("NodeCache{entries=%d, ttl=%s, maxEntries=%d, keys=%v}", total, c.ttl, c.maxEntries, keys)
	}
	return fmt.Sprintf("NodeCache{entries=%d, ttl=%s, maxEntries=%d}", total, c.ttl, c.maxEntries)
}

// Format implements fmt.Formatter to support %#v / %+v friendly output.
func (c *NodeCache) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') || f.Flag('#') {
			fmt.Fprint(f, c.Summary(0))
			return
		}
		fmt.Fprint(f, c.String())
	case 's', 'q':
		fmt.Fprint(f, c.String())
	default:
		fmt.Fprint(f, c.String())
	}
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
		entry, ok2 := val.(*CacheEntry)
		if !ok2 {
			c.entries.Delete(path)
			return nil, false
		}
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

func (c *NodeCache) GetChildren(path string) ([]os.FileInfo, bool) {
	if val, ok := c.entries.Load(path); ok {
		entry, ok2 := val.(*CacheEntry)
		if !ok2 {
			c.entries.Delete(path)
			return nil, false
		}
		if time.Now().After(entry.ExpiresAt) {
			c.entries.Delete(path)
			return nil, false
		}
		// children nil means "not cached"
		if entry.Children == nil {
			return nil, false
		}
		return entry.Children, true
	}
	return nil, false
}

func (c *NodeCache) SetChildren(path string, children []os.FileInfo) {
	var entry *CacheEntry
	if val, ok := c.entries.Load(path); ok {
		if e, ok2 := val.(*CacheEntry); ok2 {
			entry = e
		}
	}
	if entry == nil {
		entry = &CacheEntry{
			Info:      nil,
			IsDir:     true,
			Children:  children,
			ExpiresAt: time.Now().Add(c.ttl),
		}
	} else {
		entry.Children = children
		entry.IsDir = true
		entry.ExpiresAt = time.Now().Add(c.ttl)
	}
	c.entries.Store(path, entry)
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

func (c *NodeCache) Invalidate(p string) {
	if p == "" {
		return
	}
	p = filepath.Clean(p)

	c.entries.Delete(p)

	if !strings.HasSuffix(p, string(os.PathSeparator)) {
		c.entries.Delete(p + string(os.PathSeparator))
	}

	parent := filepath.Dir(p)
	if parent != p {
		c.entries.Delete(parent)
		if !strings.HasSuffix(parent, string(os.PathSeparator)) {
			c.entries.Delete(parent + string(os.PathSeparator))
		}
	}
}
