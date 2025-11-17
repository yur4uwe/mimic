//go:build linux

package fs

import (
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/fs/platform/linux"
	"github.com/mimic/internal/interfaces"
)

func New(webdavClient interfaces.WebClient, logger logger.FullLogger) FS {
	return linux.New(webdavClient, logger)
}
