package wrappers

import (
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mimic/internal/core/cache"
	"github.com/studio-b12/gowebdav"
)

// WebdavClient is a small wrapper around gowebdav.Client that
// provides the methods the filesystem expects. Implementations
// can be filled in later to use the underlying client and cache.
type WebdavClient struct {
	client *gowebdav.Client
	cache  *cache.NodeCache
}

func NewWebdavClient(client *gowebdav.Client, ttl time.Duration, maxEntries int) *WebdavClient {
	return &WebdavClient{
		client: client,
		cache: cache.NewNodeCache(
			ttl,
			maxEntries,
		),
	}
}

// Metadata and directory listing
func (w *WebdavClient) Stat(name string) (os.FileInfo, error) {
	if fi, ok := w.cache.Get(name); ok {
		return fi.Info, nil
	}

retry:
	stat, err := w.client.Stat(name)
	if err != nil {
		if !strings.HasSuffix(name, "/") && strings.Contains(err.Error(), "301") {
			name += "/"
			goto retry
		}
		return nil, err
	}

	w.cache.Set(name, w.cache.NewEntry(stat))

	return stat, nil
}

func (w *WebdavClient) ReadDir(name string) ([]os.FileInfo, error) {
	if children, ok := w.cache.GetChildren(name); ok && children != nil {
		return children, nil
	}

	infos, err := w.client.ReadDir(name)
	if err != nil {
		return nil, err
	}

	w.cache.SetChildren(name, infos)

	return infos, nil
}

// Read helpers

// Read reads the whole file and returns its bytes.
func (w *WebdavClient) Read(name string) ([]byte, error) {
	return w.client.Read(name)
}

// ReadStream returns an io.ReadCloser for streaming the whole file.
func (w *WebdavClient) ReadStream(name string) (io.ReadCloser, error) {
	return w.client.ReadStream(name)
}

// ReadStreamRange returns an io.ReadCloser for the requested range.
// The default fallback implementation reads whole file and slices the requested range.
func (w *WebdavClient) ReadStreamRange(name string, offset, length int64) (io.ReadCloser, error) {
	return w.client.ReadStreamRange(name, offset, length)
}

// Write / create / remove

// Write writes or overwrites the given file with data.
func (w *WebdavClient) Write(name string, data []byte) error {
	// use 0644 as a reasonable default mode
	if err := w.client.Write(name, data, 0644); err != nil {
		return err
	}

	// update cache entry for the file if possible
	if stat, err := w.client.Stat(name); err == nil {
		w.cache.Set(name, w.cache.NewEntry(stat))
	}

	// invalidate parent directory listing so callers see the new/updated file
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)

	return nil
}

// Create creates a new file with provided data.
// By default it can alias Write.
func (w *WebdavClient) Create(name string) error {
	if strings.HasSuffix(name, "/") {
		return &os.PathError{Op: "create", Path: name, Err: os.ErrInvalid}
	}
	return w.Write(name, []byte{})
}

// Remove deletes a file.
func (w *WebdavClient) Remove(name string) error {
	if err := w.client.Remove(name); err != nil {
		return err
	}
	// invalidate caches
	w.cache.Invalidate(name)
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	return nil
}

// Mkdir creates a directory.
func (w *WebdavClient) Mkdir(name string, mode os.FileMode) error {
	if err := w.client.Mkdir(name, mode); err != nil {
		return err
	}
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	return nil
}

// Rmdir removes a directory.
func (w *WebdavClient) Rmdir(name string) error {
	if err := w.client.RemoveAll(name); err != nil {
		return err
	}
	w.cache.Invalidate(name)
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	return nil
}

// Rename moves/renames a file or directory.
func (w *WebdavClient) Rename(oldname, newname string) error {
	err := w.client.Rename(oldname, newname, true)
	if err != nil {
		return err
	}

	w.cache.Invalidate(oldname)
	w.cache.Invalidate(newname)
	w.cache.Invalidate(path.Dir(oldname))
	w.cache.Invalidate(path.Dir(newname))

	return nil
}
