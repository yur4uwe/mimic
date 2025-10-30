//go:build windows

package fs

import (
	"fmt"
	"os"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/casters"
	"github.com/studio-b12/gowebdav"
	"github.com/winfsp/cgofuse/fuse"
)

// POSIX-like error codes
const (
	ENOENT = 2  // No such file or directory
	EIO    = 5  // I/O error
	EACCES = 13 // Permission denied
	EEXIST = 17 // File exists
	EISDIR = 21 // Is a directory
	EINVAL = 22 // Invalid argument
)

type winfspFS struct {
	fuse.FileSystemBase
	wc    *gowebdav.Client
	cache *cache.NodeCache
}

func New(webdavClient *gowebdav.Client) FS {
	return &winfspFS{
		wc: webdavClient,
		cache: cache.NewNodeCache(
			cache.DefaultTTL,
			cache.DefaultMaxEntries,
		),
	}
}

func (f *winfspFS) Mount(mountpoint string) error {
	fmt.Println("Mounting WinFSP filesystem at", mountpoint)

	host := fuse.NewFileSystemHost(f)
	if !host.Mount(mountpoint, nil) {
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (f *winfspFS) Unmount() error {
	fmt.Println("Unmounting WinFSP filesystem")
	return nil
}

func (f *winfspFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0755
		stat.Nlink = 2
		stat.Size = 0
		stat.Uid = 0
		stat.Gid = 0
		stat.Mtim = fuse.Now()
		stat.Atim = fuse.Now()
		stat.Ctim = fuse.Now()
		stat.Blksize = 4096
		stat.Birthtim = fuse.Now()
		return 0
	}

	var file os.FileInfo
	var err error

	if entry, ok := f.cache.Get(path); ok {
		file = entry.Info
		goto write_stat
	}

	fmt.Println("Getattr called for path:", path)

retry:
	file, err = f.wc.Stat(path)
	if err != nil {
		if path[len(path)-1] != '/' {
			path += "/"
			goto retry
		}
		return -EIO
	}

write_stat:
	*stat = *casters.FileInfoCast(file)

	return 0
}

func (f *winfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Println("Readdir called for path:", filepath)

	fill(".", nil, 0)
	fill("..", nil, 0)

	var entries []os.FileInfo
	if entry, ok := f.cache.Get(filepath); ok && entry.IsDir && entry.Children != nil {
		entries = entry.Children
	} else {
		items, err := f.wc.ReadDir(filepath)
		if err != nil {
			return -ENOENT
		}
		entries = items
	}

	for i, file := range entries {
		name := file.Name()
		fmt.Printf("  Entry %d: %s (dir=%v, size=%d)\n", i, name, file.IsDir(), file.Size())

		stat := casters.FileInfoCast(file)

		f.cache.Set(gowebdav.Join(filepath, name), f.cache.NewEntry(file))

		fill(name, stat, 0)
	}

	return 0
}

func (f *winfspFS) Open(path string, flags int) (int, uint64) {
	fmt.Println("Open called for path:", path)
	return 0, 0
}

func (f *winfspFS) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Read called for path:", path)
	data := "Hello, World!"
	copy(buff, data)
	return len(data)
}

func (f *winfspFS) Write(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Write called for path:", path)
	fmt.Printf("Data written: %s\n", string(buff))
	return len(buff)
}

func (f *winfspFS) Create(string, int, uint32) (int, uint64) {
	fmt.Println("Create called")
	return 0, 0
}
func (f *winfspFS) Unlink(path string) int {
	fmt.Println("Unlink called for path:", path)
	return 0
}

func (f *winfspFS) Mkdir(path string, mode uint32) int {
	fmt.Println("Mkdir called for path:", path)
	return 0
}

func (f *winfspFS) Rmdir(path string) int {
	fmt.Println("Rmdir called for path:", path)
	return 0
}

func (f *winfspFS) Rename(oldPath string, newPath string) int {
	fmt.Println("Rename called from", oldPath, "to", newPath)
	return 0
}
