package linux

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/fs/platform/linux/entries"
	"github.com/mimic/internal/interfaces"
)

type FuseFS struct {
	client interfaces.WebClient
}

func New(wc interfaces.WebClient) *FuseFS {
	return &FuseFS{
		client: wc,
	}
}

func (f *FuseFS) Mount(mountpoint string, mflags []string) error {
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mimic"),
		fuse.Subtype("mimicfs"),
	)
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

func (f *FuseFS) Unmount() error {
	return nil
}

func (f *FuseFS) Root() (fs.Node, error) {
	log.Println("Root called")
	return entries.NewNode(f.client, "/"), nil
}
