package win

import (
	"fmt"

	"github.com/mimic/internal/core/cache"
	"github.com/studio-b12/gowebdav"
	"github.com/winfsp/cgofuse/fuse"
)

type WinfspFS struct {
	fuse.FileSystemBase
	Wc    *gowebdav.Client
	Cache *cache.NodeCache
}

func (f *WinfspFS) Mount(mountpoint string) error {
	fmt.Println("Mounting WinFSP filesystem at", mountpoint)

	host := fuse.NewFileSystemHost(f)
	if !host.Mount(mountpoint, nil) {
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (f *WinfspFS) Unmount() error {
	fmt.Println("Unmounting WinFSP filesystem")
	return nil
}
