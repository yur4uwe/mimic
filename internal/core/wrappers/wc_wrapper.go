package wrappers

import (
	"bytes"
	"errors"
	"io"
	"os"
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
	return nil, errors.New("not implemented")
}

func (w *WebdavClient) ReadDir(name string) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

// Read helpers

// Read reads the whole file and returns its bytes.
func (w *WebdavClient) Read(name string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// ReadStream returns an io.ReadCloser for streaming the whole file.
func (w *WebdavClient) ReadStream(name string) (io.ReadCloser, error) {
	// default fallback implementation using Read
	data, err := w.Read(name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// ReadStreamRange returns an io.ReadCloser for the requested range.
// The default fallback implementation reads whole file and slices the requested range.
func (w *WebdavClient) ReadStreamRange(name string, offset, length int64) (io.ReadCloser, error) {
	data, err := w.Read(name)
	if err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, errors.New("invalid offset")
	}
	if offset >= int64(len(data)) {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	end := offset + length
	if end > int64(len(data)) || length <= 0 {
		end = int64(len(data))
	}
	slice := data[offset:end]
	return io.NopCloser(bytes.NewReader(slice)), nil
}

// Write / create / remove

// Write writes or overwrites the given file with data.
func (w *WebdavClient) Write(name string, data []byte) error {
	return errors.New("not implemented")
}

// Create creates a new file with provided data.
// By default it can alias Write.
func (w *WebdavClient) Create(name string, data []byte) error {
	return errors.New("not implemented")
}

// Remove deletes a file.
func (w *WebdavClient) Remove(name string) error {
	return errors.New("not implemented")
}

// Mkdir creates a directory.
func (w *WebdavClient) Mkdir(name string) error {
	return errors.New("not implemented")
}

// Rmdir removes a directory.
func (w *WebdavClient) Rmdir(name string) error {
	return errors.New("not implemented")
}

// Rename moves/renames a file or directory.
func (w *WebdavClient) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}
