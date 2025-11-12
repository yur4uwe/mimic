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
// provides the methods the filesystem expects
type WebdavClient struct {
	client *gowebdav.Client
	cache  *cache.NodeCache
}

const streamThreshold = 4 * 1024 * 1024 // 4 MB

func NewWebdavClient(client *gowebdav.Client, ttl time.Duration, maxEntries int) *WebdavClient {
	return &WebdavClient{
		client: client,
		cache: cache.NewNodeCache(
			ttl,
			maxEntries,
		),
	}
}

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

func (w *WebdavClient) Read(name string) ([]byte, error) {
	return w.client.Read(name)
}

func (w *WebdavClient) ReadStream(name string) (io.ReadCloser, error) {
	return w.client.ReadStream(name)
}

func (w *WebdavClient) ReadRange(name string, offset, length int64) (io.ReadCloser, error) {
	return w.client.ReadStreamRange(name, offset, length)
}

// commit centralizes write vs stream decision and cache invalidation.
func (w *WebdavClient) commit(name string, data []byte) error {
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	if len(data) > streamThreshold {
		if err := w.client.WriteStream(name, bytes.NewReader(data), 0644); err != nil {
			return err
		}
	} else {
		if err := w.client.Write(name, data, 0644); err != nil {
			return err
		}
	}

	// Invalidate caches: file and parent listing
	w.cache.Invalidate(name)
	parent := path.Dir(name)
	if parent == "." || parent == "" {
		parent = "/"
	}
	w.cache.Invalidate(parent)

	return nil
}

// fetch reads up to 'upto' bytes from the start of the file.
// If upto <= 0 it reads the whole file. Tries ranged stream read first, falls back to full read.
func (w *WebdavClient) fetch(name string, upto int64) ([]byte, error) {
	// normalize path
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	// read range when requested and supported
	if upto > 0 {
		if rc, err := w.client.ReadStreamRange(name, 0, upto); err == nil {
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			// Ensure returned slice length <= upto
			if int64(len(data)) > upto {
				return data[:upto], nil
			}
			return data, nil
		}
		// fallback to full read and then slice
		all, err := w.client.Read(name)
		if err != nil {
			return nil, err
		}
		if int64(len(all)) > upto {
			return all[:upto], nil
		}
		return all, nil
	}

	// upto <= 0 -> read whole file (try stream then Read)
	if rc, err := w.client.ReadStream(name); err == nil {
		defer rc.Close()
		return io.ReadAll(rc)
	}

	return w.client.Read(name)
}

func (w *WebdavClient) Write(name string, data []byte) error {
	return w.commit(name, data)
}

// WriteOffset merges incoming data at offset with existing content and commits via commit.
func (w *WebdavClient) WriteOffset(name string, data []byte, offset int64) error {
	// fetch existing prefix up to offset
	existing, err := w.fetch(name, offset)
	if err != nil {
		// if file doesn't exist and offset == 0 we can create new
		if os.IsNotExist(err) && offset == 0 {
			return w.commit(name, data)
		}
		return err
	}

	// compute resulting size
	end := offset + int64(len(data))
	var merged []byte
	if int64(len(existing)) >= end {
		// existing already covers the write region; modify in place
		merged = make([]byte, len(existing))
		copy(merged, existing)
		copy(merged[offset:], data)
	} else {
		// need to allocate bigger buffer
		merged = make([]byte, end)
		copy(merged, existing)
		copy(merged[offset:], data)
	}

	return w.commit(name, merged)
}

// Create creates a new file with provided data (empty).
func (w *WebdavClient) Create(name string) error {
	if strings.HasSuffix(name, "/") {
		return &os.PathError{Op: "create", Path: name, Err: os.ErrInvalid}
	}
	return w.commit(name, []byte{})
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

	w.cache.Invalidate(name)
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
	if err := w.client.Rename(oldname, newname, true); err != nil {
		return err
	}

	w.cache.Invalidate(oldname)
	w.cache.Invalidate(newname)

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
		// create empty file if it doesn't exist and size == 0
		if os.IsNotExist(err) && size == 0 {
			return w.Create(name)
		}
		return err
	}
	cur := fi.Size()

	// nothing to do
	if int64(cur) == size {
		return nil
	}

	// shrink
	if int64(cur) > size {
		// try to fetch exact prefix up to size
		data, err := w.fetch(name, size)
		if err != nil {
			return err
		}
		// ensure slice length equals requested size
		if int64(len(data)) > size {
			data = data[:size]
		} else if int64(len(data)) < size {
			// unexpected but pad with zeros
			padded := make([]byte, size)
			copy(padded, data)
			data = padded
		}
		return w.commit(name, data)
	}

	// extend: fetch whole existing content then pad zeros
	existing, err := w.fetch(name, 0)
	if err != nil {
		// if not exists and size > 0 create zero-filled
		if os.IsNotExist(err) {
			buf := make([]byte, size)
			return w.commit(name, buf)
		}
		return err
	}

	if int64(len(existing)) >= size {
		// already >= size (shouldn't happen due to earlier check) but handle defensively
		return w.commit(name, existing[:size])
	}

	buf := make([]byte, size)
	copy(buf, existing)
	// rest is zeroed by default
	return w.commit(name, buf)
}
