package fs

import (
	"fmt"
	"sync"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/interfaces"
	"github.com/winfsp/cgofuse/fuse"
)

type FS interface {
	Mount(mountpoint string, flags []string) error
	Unmount() error
}

type FuseFS struct {
	fuse.FileSystemBase
	client      interfaces.WebClient
	handles     sync.Map // map[uint64]*FileHandle
	logger      logger.FullLogger
	nextHandle  uint64
	host        *fuse.FileSystemHost
	mpoint      string
	bufferCache *cache.BufferCache
}

func New(webdavClient interfaces.WebClient, logger logger.FullLogger) *FuseFS {
	return &FuseFS{
		client:      webdavClient,
		logger:      logger,
		bufferCache: cache.NewBufferCache(),
	}
}

func (fs *FuseFS) Mount(mountpoint string, flags []string) error {
	fs.logger.Logf("Mounting FUSE filesystem at %s with flags: %v", mountpoint, flags)
	fs.mpoint = mountpoint
	fs.host = fuse.NewFileSystemHost(fs)
	if !fs.host.Mount(fs.mpoint, flags) {
		fs.logger.Error("Failed to mount FUSE filesystem")
		return fmt.Errorf("failed to mount FUSE filesystem")
	}

	return nil
}

func (fs *FuseFS) Unmount() error {
	fs.logger.Log("Unmounting FUSE filesystem")
	if ok := fs.host.Unmount(); !ok {
		fs.logger.Error("Failed to unmount FUSE filesystem")
		return fmt.Errorf("failed to unmount FUSE filesystem")
	}

	return nil
}
