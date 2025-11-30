package linux

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
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
func (fs *FuseFS) Mount(mountpoint string, mflags []string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.mounted {
		return fmt.Errorf("already mounted at %q", fs.mountpoint)
	}

	_, err := os.Stat(mountpoint)
	if !os.IsNotExist(err) && err != nil {
		return fmt.Errorf("cannot access mountpoint %q: %w", mountpoint, err)
	}

	fs.logger.Log("Mounting...")
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mimic"),
		fuse.Subtype("mimicfs"),
	)
	if err != nil {
		return fmt.Errorf("fuse mount failed: %w", err)
	}
	fs.logger.Log("Mounted, starting server...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		fusefs.Serve(c, fs)
	}()

	fs.logger.Log("Fuse is serving")
	<-sigChan

	// store runtime state for Unmount
	fs.mountpoint = mountpoint
	fs.conn = c
	fs.mounted = true

	fs.logger.Logf("Mounted Fuse on %s", mountpoint)
	return nil
}

// Unmount stops serving, unmounts the filesystem and releases resources.
// It is safe to call multiple times.
func (fs *FuseFS) Unmount() error {
	fs.mu.Lock()
	// capture state to operate on while unlocked for potentially blocking ops
	mounted := fs.mounted
	mp := fs.mountpoint
	conn := fs.conn
	serveErr := fs.serveErr

	fs.mounted = false
	fs.mountpoint = ""
	fs.conn = nil
	fs.serveErr = nil
	fs.mu.Unlock()

	if !mounted {
		return nil
	}

	if err := fuse.Unmount(mp); err != nil {
		// Unmount can fail when processes keep files open. Try a lazy unmount fallback,
		// and log the error so operator can take manual action (fuser/kill).
		fs.logger.Errorf("fuse.Unmount error: %v", err)
		if ex := exec.Command("fusermount3", "-uz", mp).Run(); ex != nil {
			fs.logger.Errorf("fusermount3 -uz failed: %v", ex)
		}
	}

	if conn != nil {
		_ = conn.Close()
	}

	if serveErr != nil {
		fs.logger.Errorf("fs.Serve returned error: %v", serveErr)
	}

	fs.logger.Logf("Unmounted %s", mp)
	return nil
}

func (fs *FuseFS) Root() (fusefs.Node, error) {
	fs.logger.Log("Root called")
	return entries.NewNode(fs.client, fs.logger, "/"), nil
}
