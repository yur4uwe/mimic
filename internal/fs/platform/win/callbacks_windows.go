package win

import (
	"fmt"
	"os"

	"github.com/mimic/internal/core/casters"
	"github.com/studio-b12/gowebdav"
	"github.com/winfsp/cgofuse/fuse"
)

const (
	ENOENT = fuse.ENOENT
	EIO    = fuse.EIO
)

func (f *WinfspFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
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

	if entry, ok := f.Cache.Get(path); ok {
		file = entry.Info
		goto write_stat
	}

	fmt.Println("Getattr called for path:", path)

retry:
	file, err = f.Wc.Stat(path)
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

func (f *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Println("Readdir called for path:", filepath)

	fill(".", nil, 0)
	fill("..", nil, 0)

	var entries []os.FileInfo
	if entry, ok := f.Cache.Get(filepath); ok && entry.IsDir && entry.Children != nil {
		entries = entry.Children
	} else {
		items, err := f.Wc.ReadDir(filepath)
		if err != nil {
			return -ENOENT
		}
		entries = items
	}

	for i, file := range entries {
		name := file.Name()
		fmt.Printf("  Entry %d: %s (dir=%v, size=%d)\n", i, name, file.IsDir(), file.Size())

		stat := casters.FileInfoCast(file)

		f.Cache.Set(gowebdav.Join(filepath, name), f.Cache.NewEntry(file))

		fill(name, stat, 0)
	}

	return 0
}

func (f *WinfspFS) Open(path string, flags int) (int, uint64) {
	fmt.Println("Open called for path:", path)
	return 0, 0
}

func (f *WinfspFS) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Read called for path:", path)
	data := "Hello, World!"
	copy(buff, data)
	return len(data)
}

func (f *WinfspFS) Write(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Write called for path:", path)
	fmt.Printf("Data written: %s\n", string(buff))
	return len(buff)
}

func (f *WinfspFS) Create(string, int, uint32) (int, uint64) {
	fmt.Println("Create called")
	return 0, 0
}
func (f *WinfspFS) Unlink(path string) int {
	fmt.Println("Unlink called for path:", path)
	return 0
}

func (f *WinfspFS) Mkdir(path string, mode uint32) int {
	fmt.Println("Mkdir called for path:", path)
	return 0
}

func (f *WinfspFS) Rmdir(path string) int {
	fmt.Println("Rmdir called for path:", path)
	return 0
}

func (f *WinfspFS) Rename(oldPath string, newPath string) int {
	fmt.Println("Rename called from", oldPath, "to", newPath)
	return 0
}
