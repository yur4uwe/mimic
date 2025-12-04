package wrappers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/locking"
	"github.com/mimic/internal/fs/common"
	"github.com/studio-b12/gowebdav"
)

type WebdavClient struct {
	client *gowebdav.Client
	cache  *cache.NodeCache
	lm     *locking.LockManager

	baseURL  string
	username string
	password string
}

const streamThreshold = 4 * 1024 * 1024 // 4 MB

func NewWebdavClient(cache *cache.NodeCache, baseURL, username, password string) *WebdavClient {
	client := gowebdav.NewClient(baseURL, username, password)
	fmt.Println("Trying to connect to the server...")
	if err := client.Connect(); err != nil {
		fmt.Fprintln(os.Stderr, "webdav client: couldn't connect to the server:", err)
		os.Exit(1)
	}
	fmt.Println("Server health check successful")

	return &WebdavClient{
		client:   client,
		cache:    cache,
		baseURL:  baseURL,
		username: username,
		password: password,
		lm:       locking.NewLockManager(),
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

func (w *WebdavClient) Write(name string, data []byte) error {
	return w.commit(name, data)
}

func (w *WebdavClient) WriteOffset(name string, data []byte, offset int64) error {
	existing, err := w.fetch(name)
	if err != nil {
		if common.IsNotExistErr(err) && offset == 0 {
			return w.commit(name, data)
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

	return w.commit(name, merged)
}

func (w *WebdavClient) Create(name string) error {
	if strings.HasSuffix(name, "/") {
		return &os.PathError{Op: "create", Path: name, Err: os.ErrInvalid}
	}
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.Invalidate(parent)
	return w.commit(name, []byte{})
}

func (w *WebdavClient) Remove(name string) error {
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.Invalidate(name)
	return w.client.Remove(name)
}

func (w *WebdavClient) Mkdir(name string, mode os.FileMode) error {
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.Invalidate(name)
	return w.client.Mkdir(name+"/", mode)
}

func (w *WebdavClient) Rmdir(name string) error {
	parent := path.Dir(strings.TrimRight(name, "/"))
	if parent == "." {
		parent = "/"
	}
	defer w.cache.InvalidateTree(parent + "/")
	defer w.cache.InvalidateTree(name + "/")
	return w.client.RemoveAll(name + "/")
}

func (w *WebdavClient) Rename(oldname, newname string) error {
	oldParent := path.Dir(strings.TrimRight(oldname, "/"))
	if oldParent == "." {
		oldParent = "/"
	}
	newParent := path.Dir(strings.TrimRight(newname, "/"))
	if newParent == "." {
		newParent = "/"
	}

	defer w.cache.InvalidateTree(oldname)
	defer w.cache.InvalidateTree(newname)

	// Try a custom MOVE request with a path-only Destination to avoid host/scheme mismatches
	if w.baseURL != "" {
		url := buildURL(w.baseURL, oldname)
		headers := map[string]string{
			"Destination": newname,
			"Overwrite":   "T",
		}

		code, _, err := davRequest("MOVE", url, w.username, w.password, nil, headers)
		if err == nil && code >= 200 && code < 300 {
			return nil
		}
		// Fall through to gowebdav rename on network error or non-2xx responses
	}

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
		if common.IsNotExistErr(err) && size == 0 {
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
		if common.IsNotExistErr(err) {
			buf := make([]byte, size)
			return w.commit(name, buf)
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
		return w.commit(name, existing)
	}

	if int64(len(existing)) >= size {
		// already >= size (shouldn't happen due to earlier check) but handle defensively
		return w.commit(name, existing[:size])
	}

	buf := make([]byte, size)
	copy(buf, existing)
	return w.commit(name, buf)
}

// Range-locking API used by FS layer. These are intentionally not part of
// interfaces.WebClient to avoid breaking the interface; the FS will type-assert
// to use them when present.
func (w *WebdavClient) Lock(name string, owner []byte, start, end uint64, lockType locking.LockType) error {
	return w.lm.Acquire(name, owner, start, end, lockType)
}

func (w *WebdavClient) LockWait(ctx context.Context, name string, owner []byte, start, end uint64, lockType locking.LockType) error {
	return w.lm.AcquireWait(ctx, name, owner, start, end, lockType)
}

func (w *WebdavClient) Unlock(name string, owner []byte, start, end uint64) error {
	return w.lm.Release(name, owner, start, end)
}

func (w *WebdavClient) Query(name string, start, end uint64) *locking.LockInfo {
	info, found := w.lm.Query(name, start, end)
	if !found {
		return nil
	}
	return &info
}
