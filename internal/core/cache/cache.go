package cache

import (
	"fmt"
	"os"
	"path"
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

	c.entries.Range(func(k, v any) bool {
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
			var b strings.Builder
			fmt.Fprintf(&b, "NodeCache{entries=%d, ttl=%s, maxEntries=%d}\n", func() int {
				cnt := 0
				c.entries.Range(func(_, _ interface{}) bool { cnt++; return true })
				return cnt
			}(), c.ttl, c.maxEntries)

			c.entries.Range(func(k, v interface{}) bool {
				key, _ := k.(string)
				fmt.Fprintf(&b, "Key: %s\n", key)

				entry, ok := v.(*CacheEntry)
				if !ok || entry == nil {
					fmt.Fprintln(&b, "  <invalid or nil entry>")
					return true
				}

				fmt.Fprintf(&b, "  IsDir: %v\n", entry.IsDir)
				fmt.Fprintf(&b, "  ExpiresAt: %s\n", entry.ExpiresAt.Format(time.RFC3339Nano))

				if entry.Info != nil {
					fmt.Fprintf(&b, "  Info: name=%q size=%d mode=%#o modtime=%s isDir=%v\n",
						entry.Info.Name(), entry.Info.Size(), entry.Info.Mode(), entry.Info.ModTime().Format(time.RFC3339Nano), entry.Info.IsDir())
				} else {
					fmt.Fprintln(&b, "  Info: <nil>")
				}

				if entry.Children == nil {
					fmt.Fprintln(&b, "  Children: <nil>")
				} else {
					fmt.Fprint(&b, "  Children: [")
					for i, ch := range entry.Children {
						if i > 0 {
							fmt.Fprint(&b, ", ")
						}
						if ch == nil {
							fmt.Fprint(&b, "<nil>")
						} else {
							fmt.Fprintf(&b, "%q", ch.Name())
						}
					}
					fmt.Fprintln(&b, "]")
				}

				return true
			})

			fmt.Fprint(f, b.String())
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

	c.entries.Delete(p)

	if !strings.HasSuffix(p, "/") {
		c.entries.Delete(p + "/")
	}

	parent := path.Dir(p)
	if parent != p {
		c.entries.Delete(parent)
		if !strings.HasSuffix(parent, "/") {
			c.entries.Delete(parent + "/")
		}
	}
}
