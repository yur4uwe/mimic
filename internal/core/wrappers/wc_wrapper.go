package wrappers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/mimic/internal/core/cache"
	"github.com/studio-b12/gowebdav"
)

// WebdavClient is a small wrapper around gowebdav.Client that
// provides the methods the filesystem expects
type WebdavClient struct {
	client *gowebdav.Client
	cache  *cache.NodeCache
	// base info for performing custom HTTP requests (partial PUT)
	baseURL         string
	username        string
	password        string
	allowPartialPut bool
}

const streamThreshold = 4 * 1024 * 1024 // 4 MB

func NewWebdavClient(client *gowebdav.Client, cache *cache.NodeCache, baseURL, username, password string, allowPartialPut bool) *WebdavClient {
	return &WebdavClient{
		client:          client,
		cache:           cache,
		baseURL:         baseURL,
		username:        username,
		password:        password,
		allowPartialPut: allowPartialPut,
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
	if children, ok := w.cache.GetChildren(name + "/"); ok && children != nil {
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
func (w *WebdavClient) commit(name string, offset int64, data []byte) error {
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	defer w.cache.Invalidate(name)
	// If an offset was provided, try a partial PUT using Content-Range.
	// This is non-standard but some servers (including some Apache setups)
	// accept it. If it succeeds (2xx) we return success; otherwise fall
	// back to the regular full-file write.
	if offset > 0 && w.baseURL != "" && w.allowPartialPut {
		if ok, err := w.tryPartialPut(name, offset, data); ok {
			return nil
		} else if err != nil {
			// ignore error and fall back to full write
			_ = err
		}
	}

	var err error
	if len(data) > streamThreshold {
		err = w.client.WriteStream(name, bytes.NewReader(data), 0644)
	} else {
		err = w.client.Write(name, data, 0644)
	}
	return err
}

// tryPartialPut attempts a non-standard partial PUT using Content-Range header.
// Returns (true, nil) if the server accepted the partial update (2xx),
// (false, nil) if server rejected (non-2xx), or (false, err) on network error.
func (w *WebdavClient) tryPartialPut(name string, offset int64, data []byte) (bool, error) {
	// build URL
	base := strings.TrimRight(w.baseURL, "/")
	path := strings.TrimLeft(name, "/")
	url := base + "/" + path

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return false, err
	}

	// Content-Range: bytes <start>-<end>/<total or *>
	end := offset + int64(len(data)) - 1
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", offset, end))

	if w.username != "" {
		req.SetBasicAuth(w.username, w.password)
	}

	// Use a short-lived http.Client; reusing gowebdav transport would be ideal
	// but it's not exposed from the client. Keep default settings.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, nil
}

func (w *WebdavClient) fetch(name string) ([]byte, error) {
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	// Always read the whole file (prefer streaming), then return a prefix if requested.
	if rc, err := w.client.ReadStream(name); err == nil {
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	all, err := w.client.Read(name)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func (w *WebdavClient) Write(name string, data []byte) error {
	return w.commit(name, 0, data)
}

func (w *WebdavClient) WriteOffset(name string, data []byte, offset int64) error {
	// If partial PUTs are enabled, delegate to commit (which will try partial PUT).
	if w.allowPartialPut {
		return w.commit(name, offset, data)
	}

	// Otherwise, fall back to server-agnostic behavior: fetch existing content,
	// merge the new data at the requested offset and upload the full result.
	// This preserves correct semantics on servers that don't support partial PUT.
	existing, err := w.fetch(name)
	if err != nil {
		if os.IsNotExist(err) && offset == 0 {
			return w.commit(name, 0, data)
		}
		return err
	}

	end := offset + int64(len(data))
	var merged []byte
	if int64(len(existing)) >= end {
		merged = make([]byte, len(existing))
		copy(merged, existing)
		copy(merged[offset:], data)
	} else {
		merged = make([]byte, end)
		copy(merged, existing)
		copy(merged[offset:], data)
	}

	return w.commit(name, 0, merged)
}

func (w *WebdavClient) Create(name string) error {
	if strings.HasSuffix(name, "/") {
		return &os.PathError{Op: "create", Path: name, Err: os.ErrInvalid}
	}
	// invalidate parent dir listings after create
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	return w.commit(name, 0, []byte{})
}

func (w *WebdavClient) Remove(name string) error {
	// invalidate parent dir listings after remove
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.Invalidate(name)
	return w.client.Remove(name)
}

func (w *WebdavClient) Mkdir(name string, mode os.FileMode) error {
	// invalidate parent dir listings after mkdir
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.Invalidate(name)
	return w.client.Mkdir(name+"/", mode)
}

func (w *WebdavClient) Rmdir(name string) error {
	// invalidate parent dir listings after rmdir
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.InvalidateTree(name + "/")
	return w.client.RemoveAll(name + "/")
}

func (w *WebdavClient) Rename(oldname, newname string) error {
	// invalidate parent dirs of both source and destination
	oldParent := path.Dir(strings.TrimRight(oldname, "/"))
	if oldParent == "." {
		oldParent = "/"
	}
	newParent := path.Dir(strings.TrimRight(newname, "/"))
	if newParent == "." {
		newParent = "/"
	}
	defer w.cache.InvalidateTree(oldParent + "/")
	defer w.cache.InvalidateTree(newParent + "/")

	// still invalidate trees for the entries themselves
	defer w.cache.InvalidateTree(oldname)
	defer w.cache.InvalidateTree(newname)
	return w.client.Rename(oldname, newname, true)
}

// Truncate resizes the remote file to `size`.
// Strategy:
//   - stat current size
//   - if shrinking: read range [0,size) (prefer ReadStreamRange) and PUT that slice
//   - if extending: read whole file (or available prefix), append zero bytes to requested size and PUT
func (w *WebdavClient) Truncate(name string, size int64) error {
	defer w.cache.Invalidate(name)
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

	// extend
	existing, err := w.fetch(name)
	if err != nil {
		// if not exists and size > 0 create zero-filled
		if os.IsNotExist(err) {
			buf := make([]byte, size)
			return w.commit(name, 0, buf)
		}
		return err
	}

	// shrink
	if int64(cur) > size {
		// ensure slice length equals requested size
		if int64(len(existing)) > size {
			existing = existing[:size]
		} else if int64(len(existing)) < size {
			// unexpected but pad with zeros
			padded := make([]byte, size)
			copy(padded, existing)
			existing = padded
		}
		return w.commit(name, 0, existing)
	}

	if int64(len(existing)) >= size {
		// already >= size (shouldn't happen due to earlier check) but handle defensively
		return w.commit(name, 0, existing[:size])
	}

	buf := make([]byte, size)
	copy(buf, existing)
	return w.commit(name, 0, buf)
}
