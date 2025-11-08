package win

import (
	"fmt"
	"sync"

	"github.com/mimic/internal/interfaces"
	"github.com/winfsp/cgofuse/fuse"
)

type WinfspFS struct {
	fuse.FileSystemBase
	client     interfaces.WebClient
	handles    sync.Map // map[uint64]*FileHandle
	nextHandle uint64
}

func New(webdavClient interfaces.WebClient) *WinfspFS {
	return &WinfspFS{
		client: webdavClient,
	}
}

func (f *WinfspFS) Mount(mountpoint string, flags []string) error {
	fmt.Println("Mounting WinFSP filesystem at", mountpoint)

	host := fuse.NewFileSystemHost(f)
	if !host.Mount(mountpoint, flags) {
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (f *WinfspFS) Unmount() error {
	fmt.Println("Unmounting WinFSP filesystem")
	return nil
}
