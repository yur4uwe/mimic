package fs

import (
	"os"
	"path"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/helpers"
	"github.com/winfsp/cgofuse/fuse"
)

func (fs *WinfspFS) Opendir(path string) (int, uint64) {
	fs.logger.Logf("[Opendir] path=%s", path)

	f, err := fs.client.Stat(path)
	if err != nil {
		if helpers.IsNotExistErr(err) {
			fs.logger.Errorf("[Opendir] stat: %s not found: %v; returning ENOENT", path, err)
			return -ENOENT, 0
		}
		fs.logger.Errorf("[Opendir] stat error for %s: %v; returning EIO", path, err)
		return -EIO, 0
	}

	handle := fs.NewHandle(path, casters.FileInfoCast(f), 0)
	return 0, handle
}

func (fs *WinfspFS) Releasedir(path string, fh uint64) int {
	fs.logger.Logf("[Releasedir] path=%s fh=%d", path, fh)
	fs.handles.Delete(fh)
	return 0
}

func (fs *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fs.logger.Logf("[Readdir] path=%s offset=%d fh=%d", filepath, off, fh)

	fill(".", nil, 0)
	fill("..", nil, 0)

	items, err := fs.client.ReadDir(filepath)
	if err != nil {
		return -ENOENT
	}

	for i, file := range items {
		name, err := casters.NormalizePath(path.Base(file.Name()))
		if err != nil {
			fs.logger.Errorf("[Readdir] Path normalize error for path=%s error=%v", file.Name(), err)
			continue
		}

		stat := casters.FileInfoCast(file)

		fs.logger.Logf("[ReaddirEntry] idx=%d name=%s dir=%v size=%d", i, name, file.IsDir(), file.Size())

		fill(name, stat, 0)
	}

	return 0
}

func (fs *WinfspFS) Mkdir(p string, mode uint32) int {
	fs.logger.Logf("[Mkdir] path=%s mode=%#o", p, mode)
	s, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Mkdir] Path unescape error for path=%s error=%v", p, err)
		return -EIO
	}

	err = fs.client.Mkdir(s, os.FileMode(mode))
	if err != nil {
		fs.logger.Errorf("[Mkdir] mkdir error for path=%s error=%v", s, err)
		return -EIO
	}

	return 0
}

func (fs *WinfspFS) Rmdir(path string) int {
	fs.logger.Logf("[Rmdir] path=%s", path)

	err := fs.client.Rmdir(path)
	if err != nil {
		fs.logger.Errorf("[Rmdir] rmdir error for path=%s error=%v", path, err)
		return -EIO
	}

	return 0
}
