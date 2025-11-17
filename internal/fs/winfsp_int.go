//go:build windows

package fs

import (
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/fs/platform/win"
	"github.com/mimic/internal/interfaces"
)

func New(webdavClient interfaces.WebClient, logger logger.FullLogger) FS {
	return win.New(webdavClient, logger)
}
