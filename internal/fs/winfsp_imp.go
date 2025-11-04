//go:build windows

package fs

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/interfaces"
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

const (
	DEFAULT_BLOCK_SIZE = 4096
	READ_LEN           = 1024 * 1024
)

type winfspFS struct {
	fuse.FileSystemBase
	clent      interfaces.WebClient
	handles    sync.Map // map[uint64]*FileHandle
	nextHandle uint64
}

type openedFile struct {
	path  string
	flags int
	size  int64
}

func New(webdavClient interfaces.WebClient) FS {
	return &winfspFS{
		clent: webdavClient,
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

	fmt.Println("Getattr called for path:", path)

retry:
	file, err = f.clent.Stat(path)
	if err != nil {
		if path[len(path)-1] != '/' {
			path += "/"
			goto retry
		}
		return -EIO
	}

	*stat = *casters.FileInfoCast(file)

	return 0
}

func (f *winfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Println("Readdir called for path:", filepath)

	fill(".", nil, 0)
	fill("..", nil, 0)

	var entries []os.FileInfo

	items, err := f.clent.ReadDir(filepath)
	if err != nil {
		return -ENOENT
	}
	entries = items

	for i, file := range entries {
		name := file.Name()
		fmt.Printf("  Entry %d: %s (dir=%v, size=%d)\n", i, name, file.IsDir(), file.Size())

		stat := casters.FileInfoCast(file)

		fill(name, stat, 0)
	}

	return 0
}

func (f *winfspFS) Open(path string, flags int) (int, uint64) {
	fmt.Println("Open called for path:", path)

	f.clent.Stat(path)

	return 0, 0
}

func (f *winfspFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Println("Read called for path:", path)

	toRead := len(buffer)
	rc, err := f.clent.ReadStreamRange(path, offset, int64(toRead))
	if err != nil {
		return -EIO
	}
	defer rc.Close()

	n, err := io.ReadFull(rc, buffer)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return n
	} else if err != nil {
		return -EIO
	}

	return n
}

func (f *winfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Println("Write called for path:", path)
	fmt.Printf("Data written: %s\n", string(buffer))
	return len(buffer)
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

func (f *winfspFS) Release(path string, file_handle uint64) int {
	fmt.Println("Release called for path:", path, "handle:", file_handle)
	f.handles.Delete(file_handle)
	return 0
}
