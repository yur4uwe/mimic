package cache_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	cache "github.com/mimic/internal/core/cache"
)

// fakeFileInfo implements os.FileInfo for tests.
type fakeFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f *fakeFileInfo) IsDir() bool        { return f.isDir }
func (f *fakeFileInfo) Sys() interface{}   { return nil }

func makeFI(name string, isDir bool, size int64) os.FileInfo {
	return &fakeFileInfo{
		name:    name,
		size:    size,
		mode:    0644,
		modTime: time.Now(),
		isDir:   isDir,
	}
}

func dumpAndFail(t *testing.T, c *cache.NodeCache, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Fatalf("%s\n\nCACHE FORMAT (%%#v):\n%s\n",
		msg, fmt.Sprintf("%#v", c))
}

func TestSetGetAndChildren(t *testing.T) {
	c := cache.NewNodeCache(1*time.Second, 100)

	fi := makeFI("file.txt", false, 123)
	ent := c.NewEntry(fi)
	c.Set("/file.txt", ent)

	got, ok := c.Get("/file.txt")
	if !ok {
		dumpAndFail(t, c, "expected entry to be present")
	}
	if got.Info.Name() != "file.txt" {
		dumpAndFail(t, c, "unexpected info name: %q", got.Info.Name())
	}

	child1 := makeFI("a", false, 1)
	child2 := makeFI("b", true, 0)
	c.SetChildren("/dir", []os.FileInfo{child1, child2})

	children, ok := c.GetChildren("/dir")
	if !ok {
		dumpAndFail(t, c, "expected children to be present")
	}
	if len(children) != 2 || children[0].Name() != "a" || children[1].Name() != "b" {
		dumpAndFail(t, c, "unexpected children: %#v", children)
	}
}

func TestExpiration(t *testing.T) {
	c := cache.NewNodeCache(50*time.Millisecond, 100)
	fi := makeFI("tmp", false, 1)
	c.Set("/tmp", c.NewEntry(fi))

	if _, ok := c.Get("/tmp"); !ok {
		dumpAndFail(t, c, "entry should exist immediately after set")
	}

	time.Sleep(80 * time.Millisecond)

	if _, ok := c.Get("/tmp"); ok {
		dumpAndFail(t, c, "entry should have expired")
	}
}

func TestInvalidateAndParentRemoval(t *testing.T) {
	c := cache.NewNodeCache(time.Second, 100)

	fi := makeFI("child", false, 1)
	c.Set("/a/b", c.NewEntry(fi))
	c.Set("/a", c.NewEntry(makeFI("a", true, 0)))

	c.Invalidate("/a/b")

	if _, ok := c.Get("/a/b"); ok {
		dumpAndFail(t, c, "/a/b should be deleted after Invalidate")
	}
	if _, ok := c.Get("/a"); ok {
		dumpAndFail(t, c, "/a parent should be deleted by Invalidate")
	}
}

func TestInvalidateTree(t *testing.T) {
	c := cache.NewNodeCache(time.Second, 100)

	c.Set("/x/y", c.NewEntry(makeFI("y", true, 0)))
	c.Set("/x/y/z", c.NewEntry(makeFI("z", false, 0)))
	c.Set("/x/other", c.NewEntry(makeFI("other", false, 0)))

	c.InvalidateTree("/x/y")

	if _, ok := c.Get("/x/y"); ok {
		dumpAndFail(t, c, "/x/y should be removed by InvalidateTree")
	}
	if _, ok := c.Get("/x/y/z"); ok {
		dumpAndFail(t, c, "/x/y/z should be removed by InvalidateTree")
	}
	if _, ok := c.Get("/x/other"); !ok {
		dumpAndFail(t, c, "/x/other should remain after InvalidateTree")
	}
}

func TestSummaryIncludesKeys(t *testing.T) {
	c := cache.NewNodeCache(time.Second, 100)
	c.Set("/k1", c.NewEntry(makeFI("k1", false, 0)))
	c.Set("/k2", c.NewEntry(makeFI("k2", false, 0)))

	s := c.Summary(10)
	if !strings.Contains(s, "/k1") || !strings.Contains(s, "/k2") {
		dumpAndFail(t, c, "summary did not include expected keys: %s", s)
	}

	_ = c.String()
}
