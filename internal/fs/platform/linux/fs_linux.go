package linux

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

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
	lockFile   *os.File
	conn       *fuse.Conn
	serveErr   chan error
	mounted    bool
}

func New(wc interfaces.WebClient, logger logger.FullLogger) *FuseFS {
	return &FuseFS{
		client: wc,
		logger: logger,
	}
}

func lockMountpoint(mountpoint string) (*os.File, error) {
	lockPath := filepath.Join(mountpoint, ".mimic.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("mountpoint busy or locked: %w", err)
	}
	return f, nil
}

func unlockMountpoint(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
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

	// acquire lock to prevent concurrent mounts on the same directory
	lockFile, err := lockMountpoint(mountpoint)
	if err != nil {
		return fmt.Errorf("cannot lock mountpoint %q: %w", mountpoint, err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mimic"),
		fuse.Subtype("mimicfs"),
	)
	if err != nil {
		_ = lockFile.Close()
		return fmt.Errorf("fuse mount failed: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- fs.Serve(c, f)
	}()

	// store runtime state for Unmount
	f.mountpoint = mountpoint
	f.lockFile = lockFile
	f.conn = c
	f.serveErr = serveErr
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
	lockFile := f.lockFile
	conn := f.conn
	serveErr := f.serveErr

	f.mounted = false
	f.mountpoint = ""
	f.lockFile = nil
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
		if err := <-serveErr; err != nil {
			log.Printf("fs.Serve returned error: %v", err)
		}
	}

	unlockMountpoint(lockFile)

	fmt.Println("Unmounted", mp)
	return nil
}

func (f *FuseFS) Root() (fs.Node, error) {
	f.logger.Log("Root called")
	return entries.NewNode(f.client, f.logger, "/"), nil
}
