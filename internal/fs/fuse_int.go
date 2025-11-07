//go:build linux || darwin

package fs

import (
	"github.com/mimic/internal/fs/platform/linux"
	"github.com/studio-b12/gowebdav"
)

func New(webdavClient *gowebdav.Client) FS {
	return &linux.FuseFS{
		Wc: webdavClient,
	}
}
