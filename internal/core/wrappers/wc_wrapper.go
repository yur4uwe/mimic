package wrappers

import (
	"bytes"
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
		if !strings.HasSuffix(name, "/") && strings.Contains(err.Error(), "200") {
			name += "/"
			goto retry
		}
		return nil, err
	}

	w.cache.Set(name, w.cache.NewEntry(stat))

	return stat, nil
}

func (w *WebdavClient) ReadDir(name string) ([]os.FileInfo, error) {
	if !strings.HasSuffix(name, "/") {
		name += "/"
	}

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

func (w *WebdavClient) Read(name string) ([]byte, error) {
	return w.client.Read(name)
}

func (w *WebdavClient) ReadStream(name string) (io.ReadCloser, error) {
	return w.client.ReadStream(name)
}

func (w *WebdavClient) commit(name string, data []byte) error {
	// normalize name as other methods do
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	if len(data) > streamThreshold {
		// stream for large payloads to avoid copying into library buffers
		if err := w.client.WriteStream(name, bytes.NewReader(data), 0644); err != nil {
			return err
		}
	} else {
		if err := w.client.Write(name, data, 0644); err != nil {
			return err
		}
	}

	// refresh cache for file
	if stat, err := w.client.Stat(name); err == nil {
		w.cache.Set(name, w.cache.NewEntry(stat))
	}

	// invalidate parent listing and the file entry
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	w.cache.Invalidate(name)

	return nil
}

// Write writes or overwrites the given file with data.
func (w *WebdavClient) Write(name string, data []byte) error {
	return w.commit(name, data)
}

// WriteStream writes or overwrites the given file with data from the stream.
func (w *WebdavClient) WriteStream(name string, data io.Reader) error {
	if err := w.client.WriteStream(name, data, 0644); err != nil {
		return err
	}

	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	w.cache.Invalidate(name)

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

	w.cache.Invalidate(name)
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)
	return nil
}

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

// Truncate resizes the remote file to `size`.
// Strategy:
//   - stat current size
//   - if shrinking: read range [0,size) (prefer ReadStreamRange) and PUT that slice
//   - if extending: read whole file (or available prefix), append zero bytes to requested size and PUT
func (w *WebdavClient) Truncate(name string, size int64) error {
	// normalize
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	// get current size
	fi, err := w.client.Stat(name)
	if err != nil {
		// if file doesn't exist and size==0 create empty file
		if os.IsNotExist(err) {
			if size == 0 {
				if err := w.Create(name); err == nil {
					// Write already invalidates cache; return nil
					return nil
				}
			}
			return err
		}
		return err
	}
	cur := fi.Size()

	// nothing to do
	if int64(cur) == size {
		return nil
	}

	// helper to commit bytes using streaming when beneficial
	commit := func(data []byte) error {
		// prefer streaming write for large payloads to avoid huge memory duplication in client libraries
		if len(data) > streamThreshold { // threshold: 4 MiB
			// use WriteStream if available
			if err := w.client.WriteStream(name, bytes.NewReader(data), 0644); err != nil {
				return err
			}
			// update cache after successful upload
			if stat, err := w.client.Stat(name); err == nil {
				w.cache.Set(name, w.cache.NewEntry(stat))
			}
			// invalidate parent listing
			parent := path.Dir(name)
			if parent == "." || parent == "" {
				parent = "/"
			}
			w.cache.Invalidate(parent)
			return nil
		}
		// small payload — reuse existing Write which also updates cache
		return w.Write(name, data)
	}

	// shrink
	if int64(cur) > size {
		// attempt ranged read
		if rc, err := w.client.ReadStreamRange(name, 0, size); err == nil {
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return err
			}
			return commit(data)
		}
		// fallback to full read + slice
		all, err := w.client.Read(name)
		if err != nil {
			return err
		}
		if int64(len(all)) < size {
			// unexpected: treat as extend
			size = int64(len(all))
		}
		return commit(all[:size])
	}

	// extend
	// read existing content (stream or whole)
	var existing []byte
	if rc, err := w.client.ReadStreamRange(name, 0, int64(cur)); err == nil {
		defer rc.Close()
		existing, err = io.ReadAll(rc)
		if err != nil {
			return err
		}
	} else {
		all, err := w.client.Read(name)
		if err != nil {
			return err
		}
		existing = all
	}

	// create new buffer sized to `size`, copy existing and leave rest zeros
	buf := make([]byte, size)
	copy(buf, existing)

	return commit(buf)
}

const streamThreshold = 4 * 1024 * 1024 // 4 MiB, tune as needed
