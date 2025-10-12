//go:build linux || darwin

package fs

import (
	"fmt"

	"bazil.org/fuse"
	"github.com/mimic/internal/core/webdav"
)

type fuseFS struct {
	wc *webdav.Client
}

func New(webdavClient *webdav.Client) FS {
	return &fuseFS{
		wc: webdavClient,
	}
}

func (f *fuseFS) Mount(mountpoint string) error {
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return fmt.Errorf("fuse mount failed: %w", err)
	}
	defer c.Close()
	fmt.Println("Mounted on", mountpoint)
	<-make(chan struct{}) // block for demo
	return nil
}

func (f *fuseFS) Unmount() error {
	fmt.Println("Unmounting FUSE filesystem")
	return nil
}
