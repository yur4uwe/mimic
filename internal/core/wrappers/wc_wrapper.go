package wrappers

import (
	"bytes"
	"fmt"
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
	// use 0777 as a reasonable default mode
	if err := w.client.Write(name, data, 0777); err != nil {
		return err
	}

	// update cache entry for the file if possible
	if stat, err := w.Stat(name); err == nil {
		w.cache.Set(name, w.cache.NewEntry(stat))
	} else {
		fmt.Println("Error:", err)
	}

	// invalidate parent directory listing so callers see the new/updated file
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)

	return nil
}

// WriteStream writes or overwrites the given file with data from the stream.
func (w *WebdavClient) WriteStream(name string, data io.Reader) error {
	return w.client.WriteStream(name, data, 0644)
}

// WriteStreamRange writes or overwrites a range of the given file with data from the stream.
func (w *WebdavClient) WriteStreamRange(name string, data io.Reader, offset int64) error {
	// Many WebDAV clients (or their underlying HTTP helpers) expect the
	// Content-Length header to match the provided reader exactly. Some
	// implementations used by tests/clients will set ContentLength to the
	// existing file size which causes a mismatch when writing a range that
	// extends the file. To keep semantics correct and portable, implement a
	// safe fallback: read the incoming stream into memory, merge it into the
	// existing file contents at `offset`, and PUT the resulting full file.
	// This is less efficient than a true ranged upload but avoids ContentLength
	// mismatches and keeps behavior correct for small/typical writes.

	// Read incoming chunk
	chunk, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	// Normalize name
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	// Get current size (if file exists)
	var curSize int64
	if fi, err := w.client.Stat(name); err == nil {
		curSize = fi.Size()
	} else {
		// if file doesn't exist, treat curSize as 0
		curSize = 0
	}

	end := offset + int64(len(chunk))
	newSize := curSize
	if end > newSize {
		newSize = end
	}

	// Build new buffer containing existing content with chunk applied at offset
	buf := make([]byte, newSize)

	// Attempt ranged read of existing data
	if curSize > 0 {
		if rc, err := w.client.ReadStreamRange(name, 0, curSize); err == nil {
			defer rc.Close()
			if _, err := io.ReadFull(rc, buf[:curSize]); err != nil {
				// fallback to full read
				if all, err := w.client.Read(name); err == nil {
					copy(buf, all)
				} else {
					return err
				}
			}
		} else {
			// fallback to full read
			if all, err := w.client.Read(name); err == nil {
				copy(buf, all)
			} else {
				return err
			}
		}
	}

	// apply chunk at offset
	copy(buf[offset:], chunk)

	// Commit: for large payloads use streaming write to avoid extra copy in some clients
	if len(buf) > 4*1024*1024 {
		if err := w.client.WriteStream(name, bytes.NewReader(buf), 0644); err != nil {
			return err
		}
		// update cache
		if stat, err := w.client.Stat(name); err == nil {
			w.cache.Set(name, w.cache.NewEntry(stat))
		}
	} else {
		if err := w.Write(name, buf); err != nil {
			return err
		}
	}

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
				if err := w.Write(name, []byte{}); err == nil {
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
		if len(data) > 4*1024*1024 { // threshold: 4 MiB
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
