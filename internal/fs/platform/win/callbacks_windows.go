package win

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

const (
	ENOENT = fuse.ENOENT
	EIO    = fuse.EIO
)

const (
	DEFAULT_BLOCK_SIZE = 4096
	READ_LEN           = 1024 * 1024
)

type openedFile struct {
	path  string
	flags int
	size  int64
}

func (f *WinfspFS) NewHandle(path string, flags int, size int64) uint64 {
	file_handle := atomic.AddUint64(&f.nextHandle, 1)
	f.handles.Store(file_handle, &openedFile{
		path:  path,
		flags: flags,
		size:  size,
	})
	return file_handle
}

func (f *WinfspFS) GetHandle(handle uint64) (*openedFile, bool) {
	file, ok := f.handles.Load(handle)
	if !ok {
		return nil, false
	}
	return file.(*openedFile), true
}

func (f *WinfspFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 00777
		stat.Nlink = 2
		stat.Size = 0
		stat.Uid = 1000
		stat.Gid = 1000
		stat.Mtim = fuse.Now()
		stat.Atim = fuse.Now()
		stat.Ctim = fuse.Now()
		stat.Blksize = 4096
		stat.Birthtim = fuse.Now()
		return 0
	}

	fmt.Println("Getattr called for path:", path)

	file, err := f.client.Stat(path)
	if err != nil {
		return -EIO
	}

	*stat = *casters.FileInfoCast(file)

	return 0
}

func (f *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Println("Readdir called for path:", filepath)

	fill(".", nil, 0)
	fill("..", nil, 0)

	items, err := f.client.ReadDir(filepath)
	if err != nil {
		return -ENOENT
	}

	for i, file := range items {
		name := file.Name()
		fmt.Printf("  Entry %d: %s (dir=%v, size=%d)\n", i, name, file.IsDir(), file.Size())

		stat := casters.FileInfoCast(file)

		fill(name, stat, 0)
	}

	return 0
}

func (f *WinfspFS) Open(path string, flags int) (int, uint64) {

	fi, err := f.client.Stat(path)
	if err != nil {
		return -EIO, 0
	}

	if fi.IsDir() && (flags&fuse.O_WRONLY != 0 || flags&fuse.O_RDWR != 0) {
		EISDIR := 1
		return -EISDIR, 0
	}

	handle := f.NewHandle(path, flags, fi.Size())

	fmt.Println("Open called for path:", path, "with flags:", flags, "handle:", handle)

	return 0, handle
}

func (f *WinfspFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Println("Read called for path:", path)

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -EIO
	}

	if offset >= file.size {
		return 0 // EOF
	}

	toRead := len(buffer)
	rc, err := f.client.ReadStreamRange(file.path, offset, int64(toRead))
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

func (f *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Println("Write called for path:", path)
	fmt.Printf("Data written: %s\n", string(buffer))

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -EIO
	}

	if offset >= file.size {
		return 0 // EOF
	}

	err := f.client.Write(file.path, buffer)
	if err != nil {
		return -EIO
	}

	return len(buffer)
}

func (f *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fmt.Println("Create called for path:", path, "with mode:", mode, "and flags:", flags)

	err := f.client.Create(path)
	if err != nil {
		return -EIO, 0
	}

	return 0, f.NewHandle(path, flags, 0)
}

func (f *WinfspFS) Unlink(path string) int {
	fmt.Println("Unlink called for path:", path)
	return 0
}

func (f *WinfspFS) Mkdir(path string, mode uint32) int {
	fmt.Println("Mkdir called for path:", path)

	err := f.client.Mkdir(path, os.FileMode(mode))
	if err != nil {
		return -EIO
	}

	return 0
}

func (f *WinfspFS) Rmdir(path string) int {
	fmt.Println("Rmdir called for path:", path)

	err := f.client.Rmdir(path)
	if err != nil {
		return -EIO
	}

	return 0
}

func (f *WinfspFS) Rename(oldPath string, newPath string) int {
	fmt.Println("Rename called from", oldPath, "to", newPath)

	err := f.client.Rename(oldPath, newPath)
	if err != nil {
		return -EIO
	}

	return 0
}

func (f *WinfspFS) Release(path string, file_handle uint64) int {
	fmt.Println("Release called for path:", path, "handle:", file_handle)
	f.handles.Delete(file_handle)
	return 0
}
