package linux

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

func (f *FuseFS) Mount(mountpoint string, mflags []string) error {
	// acquire lock to prevent concurrent mounts on the same directory
	lockFile, err := lockMountpoint(mountpoint)
	if err != nil {
		return fmt.Errorf("cannot lock mountpoint %q: %w", mountpoint, err)
	}
	defer unlockMountpoint(lockFile)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mimic"),
		fuse.Subtype("mimicfs"),
	)
	if err != nil {
		return fmt.Errorf("fuse mount failed: %w", err)
	}
	// ensure connection closed on exit
	defer c.Close()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- fs.Serve(c, f)
	}()

	fmt.Println("Mounted Fuse on", mountpoint)

	// wait for interrupt/termination
	sigcatcher := make(chan os.Signal, 1)
	signal.Notify(sigcatcher, syscall.SIGINT, syscall.SIGTERM)
	<-sigcatcher

	// begin shutdown: ask kernel to unmount
	fmt.Println("Unmounting", mountpoint)
	if err := fuse.Unmount(mountpoint); err != nil {
		// Unmount can fail when processes keep files open. Try a lazy unmount fallback,
		// and log the error so operator can take manual action (fuser/kill).
		log.Printf("fuse.Unmount error: %v", err)
		// best-effort attempt with fusermount (may not exist on all systems)
		if ex := exec.Command("fusermount3", "-uz", mountpoint).Run(); ex != nil {
			log.Printf("fusermount3 -uz failed: %v", ex)
		}
	}

	_ = c.Close()

	if err := <-serveErr; err != nil {
		log.Printf("fs.Serve returned error: %v", err)
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
