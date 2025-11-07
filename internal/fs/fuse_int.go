//go:build linux || darwin

package fs

import (
	"github.com/mimic/internal/fs/platform/linux"
	"github.com/mimic/internal/interfaces"
)

func New(webdavClient interfaces.WebClient) FS {
	return linux.New(webdavClient)
}
