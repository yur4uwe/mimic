//go:build windows

package fs

import (
	"github.com/mimic/internal/fs/platform/win"
	"github.com/mimic/internal/interfaces"
)

func New(webdavClient interfaces.WebClient) FS {
	return win.New(webdavClient)
}
