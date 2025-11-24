package win

import (
	"fmt"
	"sync"

	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/interfaces"
	"github.com/winfsp/cgofuse/fuse"
)

type WinfspFS struct {
	fuse.FileSystemBase
	client     interfaces.WebClient
	handles    sync.Map // map[uint64]*FileHandle
	logger     logger.FullLogger
	nextHandle uint64
}

func New(webdavClient interfaces.WebClient, logger logger.FullLogger) *WinfspFS {
	return &WinfspFS{
		client: webdavClient,
		logger: logger,
	}
}

func (fs *WinfspFS) Mount(mountpoint string, flags []string) error {
	fmt.Println("Mounting WinFSP filesystem at", mountpoint)

	host := fuse.NewFileSystemHost(fs)
	if !host.Mount(mountpoint, flags) {
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (fs *WinfspFS) Unmount() error {
	fmt.Println("Unmounting WinFSP filesystem")
	return nil
}
