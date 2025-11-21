package linux

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/fs/platform/linux/entries"
	"github.com/mimic/internal/interfaces"
)

type FuseFS struct {
	client interfaces.WebClient
	logger logger.FullLogger

	// runtime mount state:
	mu         sync.Mutex
	mountpoint string
	conn       *fuse.Conn
	serveErr   error
	mounted    bool
}

func New(wc interfaces.WebClient, logger logger.FullLogger) *FuseFS {
	return &FuseFS{
		client: wc,
		logger: logger,
	}
}

// Mount creates the FUSE mount and starts fs.Serve in the background.
// It returns once the mount and serve goroutine are started. Call Unmount()
// to stop serving and clean up.
func (f *FuseFS) Mount(mountpoint string, mflags []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.mounted {
		return fmt.Errorf("already mounted at %q", f.mountpoint)
	}

	_, err := os.Stat(mountpoint)
	if !os.IsNotExist(err) && err != nil {
		return fmt.Errorf("cannot access mountpoint %q: %w", mountpoint, err)
	}

	fmt.Println("Mounting...")
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mimic"),
		fuse.Subtype("mimicfs"),
	)
	if err != nil {
		return fmt.Errorf("fuse mount failed: %w", err)
	}
	fmt.Println("Mounted, starting server...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		fs.Serve(c, f)
	}()

	fmt.Println("Fuse is serving")
	<-sigChan

	// store runtime state for Unmount
	f.mountpoint = mountpoint
	f.conn = c
	f.mounted = true

	fmt.Println("Mounted Fuse on", mountpoint)
	return nil
}

// Unmount stops serving, unmounts the filesystem and releases resources.
// It is safe to call multiple times.
func (f *FuseFS) Unmount() error {
	f.mu.Lock()
	// capture state to operate on while unlocked for potentially blocking ops
	mounted := f.mounted
	mp := f.mountpoint
	conn := f.conn
	serveErr := f.serveErr

	f.mounted = false
	f.mountpoint = ""
	f.conn = nil
	f.serveErr = nil
	f.mu.Unlock()

	if !mounted {
		return nil
	}

	if err := fuse.Unmount(mp); err != nil {
		// Unmount can fail when processes keep files open. Try a lazy unmount fallback,
		// and log the error so operator can take manual action (fuser/kill).
		log.Printf("fuse.Unmount error: %v", err)
		if ex := exec.Command("fusermount3", "-uz", mp).Run(); ex != nil {
			log.Printf("fusermount3 -uz failed: %v", ex)
		}
	}

	if conn != nil {
		_ = conn.Close()
	}

	if serveErr != nil {
		log.Printf("fs.Serve returned error: %v", serveErr)
	}

	fmt.Println("Unmounted", mp)
	return nil
}

func (f *FuseFS) Root() (fs.Node, error) {
	f.logger.Log("Root called")
	return entries.NewNode(f.client, f.logger, "/"), nil
}
