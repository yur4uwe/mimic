//go:build linux || darwin

package fs

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/fs/entries"
	"github.com/studio-b12/gowebdav"
)

type fuseFS struct {
	wc *gowebdav.Client
}

func New(webdavClient *gowebdav.Client) FS {
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

	go func() {
		if err := fs.Serve(c, f); err != nil {
			log.Printf("fs.Serve error: %v", err)
		}
	}()

	fmt.Println("Mounted Fuse on", mountpoint)

	sigcatcher := make(chan os.Signal, 1)
	signal.Notify(sigcatcher, syscall.SIGINT, syscall.SIGTERM)
	<-sigcatcher

	if err := fuse.Unmount(mountpoint); err != nil {
		return fmt.Errorf("unmount failed: %w", err)
	}
	return nil
}

func (f *fuseFS) Unmount() error {
	return nil
}

func (f *fuseFS) Root() (fs.Node, error) {
	return entries.NewNode(f.wc, "/"), nil
}
