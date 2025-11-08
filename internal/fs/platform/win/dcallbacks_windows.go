package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (f *WinfspFS) Opendir(path string) (int, uint64) {
	fmt.Println("Opendir called for path:", path)

	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	d, err := f.client.Stat(path)
	if err != nil || !d.IsDir() {
		return -ENOENT, 0
	}

	if !d.IsDir() {
		return -EIO, 0
	}

	handle := f.NewHandle(path, os.O_RDONLY, 0)
	return 0, handle
}

func (f *WinfspFS) Releasedir(path string, dir_handle uint64) int {
	fmt.Println("Releasedir called for path:", path)
	f.handles.Delete(dir_handle)
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
