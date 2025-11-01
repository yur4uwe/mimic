//go:build windows

package fs

import (
	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/fs/platform/win"
	"github.com/studio-b12/gowebdav"
)

func New(webdavClient *gowebdav.Client) FS {
	return &win.WinfspFS{
		Wc: webdavClient,
		Cache: cache.NewNodeCache(
			cache.DefaultTTL,
			cache.DefaultMaxEntries,
		),
	}
}
