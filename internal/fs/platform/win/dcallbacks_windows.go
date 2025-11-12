package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (f *WinfspFS) Opendir(path string) (int, uint64) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	d, err := f.client.Stat(path)
	if err != nil || !d.IsDir() {
		return -fuse.ENOENT, 0
	}

	if !d.IsDir() {
		return -fuse.EIO, 0
	}

	handle := f.NewHandle(path, casters.FileInfoCast(d))
	fmt.Printf("[log] (Opendir): returning handle=%d path=%s\n", handle, path)
	return 0, handle
}

func (f *WinfspFS) Releasedir(path string, dir_handle uint64) int {
	fmt.Printf("[log] (Releasedir): path=%s handle=%d\n", path, dir_handle)
	f.handles.Delete(dir_handle)
	return 0
}

func (f *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Printf("[log] (Readdir): path=%s off=%d fh=%d\n", filepath, off, fh)

	fill(".", nil, 0)
	fill("..", nil, 0)

	items, err := f.client.ReadDir(filepath)
	if err != nil {
		return -fuse.ENOENT
	}

	for i, file := range items {
		name := file.Name()

		stat := casters.FileInfoCast(file)

		fmt.Printf("[log] (ReaddirEntry): idx=%d name=%s dir=%v size=%d\n", i, name, file.IsDir(), file.Size())

		fill(name, stat, 0)
	}

	return 0
}

func (f *WinfspFS) Mkdir(path string, mode uint32) int {
	fmt.Printf("[log] (Mkdir): path=%s mode=%#o\n", path, mode)

	err := f.client.Mkdir(path, os.FileMode(mode))
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Rmdir(path string) int {
	fmt.Printf("[log] (Rmdir): path=%s\n", path)

	err := f.client.Rmdir(path)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}
