package win

import (
	"os"
	"path"
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
	f.logger.Logf("[log] (Opendir): returning handle=%d path=%s", handle, path)
	return 0, handle
}

func (f *WinfspFS) Releasedir(path string, dir_handle uint64) int {
	f.logger.Logf("[log] (Releasedir): path=%s handle=%d", path, dir_handle)
	f.handles.Delete(dir_handle)
	return 0
}

func (f *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	f.logger.Logf("[log] (Readdir): path=%s off=%d fh=%d", filepath, off, fh)

	fill(".", nil, 0)
	fill("..", nil, 0)

	items, err := f.client.ReadDir(filepath)
	if err != nil {
		return -fuse.ENOENT
	}

	for i, file := range items {
		name, err := casters.NormalizePath(path.Base(file.Name()))
		if err != nil {
			f.logger.Errorf("[Readdir] Path normalize error for path=%s error=%v", file.Name(), err)
			continue
		}

		stat := casters.FileInfoCast(file)

		f.logger.Logf("[log] (ReaddirEntry): idx=%d name=%s dir=%v size=%d\n", i, name, file.IsDir(), file.Size())

		fill(name, stat, 0)
	}

	return 0
}

func (f *WinfspFS) Mkdir(p string, mode uint32) int {
	f.logger.Logf("[log] (Mkdir): path=%s mode=%#o", p, mode)
	s, err := casters.NormalizePath(p)
	if err != nil {
		f.logger.Errorf("[Mkdir] Path unescape error for path=%s error=%v", p, err)
		return -fuse.EIO
	}

	err = f.client.Mkdir(s, os.FileMode(mode))
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Rmdir(path string) int {
	f.logger.Logf("[log] (Rmdir): path=%s", path)

	err := f.client.Rmdir(path)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}
