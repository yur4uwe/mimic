package fs

import (
	"fmt"
	"sync"

	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/interfaces"
	"github.com/winfsp/cgofuse/fuse"
)

type FS interface {
	Mount(mountpoint string, flags []string) error
	Unmount() error
}

type WinfspFS struct {
	fuse.FileSystemBase
	client     interfaces.WebClient
	handles    sync.Map // map[uint64]*FileHandle
	logger     logger.FullLogger
	nextHandle uint64
	host       *fuse.FileSystemHost
	mpoint     string
}

func New(webdavClient interfaces.WebClient, logger logger.FullLogger) *WinfspFS {
	return &WinfspFS{
		client: webdavClient,
		logger: logger,
	}
}

func (fs *WinfspFS) Mount(mountpoint string, flags []string) error {
	fs.logger.Logf("Mounting WinFSP filesystem at %s with flags: %v", mountpoint, flags)
	fs.mpoint = mountpoint
	fs.host = fuse.NewFileSystemHost(fs)
	if !fs.host.Mount(fs.mpoint, flags) {
		fs.logger.Error("Failed to mount WinFSP filesystem")
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (fs *WinfspFS) Unmount() error {
	fs.logger.Log("Unmounting WinFSP filesystem")
	if ok := fs.host.Unmount(); !ok {
		fs.logger.Error("Failed to unmount WinFSP filesystem")
		return fmt.Errorf("failed to unmount winfsp filesystem")
	}

	return nil
}
