package win

import (
	"os"
	"path"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (fs *WinfspFS) Readdir(filepath string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fs.logger.Logf("[log] (Readdir): path=%s off=%d fh=%d", filepath, off, fh)

	fill(".", nil, 0)
	fill("..", nil, 0)

	items, err := fs.client.ReadDir(filepath)
	if err != nil {
		return -fuse.ENOENT
	}

	for i, file := range items {
		name, err := casters.NormalizePath(path.Base(file.Name()))
		if err != nil {
			fs.logger.Errorf("[Readdir] Path normalize error for path=%s error=%v", file.Name(), err)
			continue
		}

		stat := casters.FileInfoCast(file)

		fs.logger.Logf("[log] (ReaddirEntry): idx=%d name=%s dir=%v size=%d\n", i, name, file.IsDir(), file.Size())

		fill(name, stat, 0)
	}

	return 0
}

func (fs *WinfspFS) Mkdir(p string, mode uint32) int {
	fs.logger.Logf("[log] (Mkdir): path=%s mode=%#o", p, mode)
	s, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Mkdir] Path unescape error for path=%s error=%v", p, err)
		return -fuse.EIO
	}

	err = fs.client.Mkdir(s, os.FileMode(mode))
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (fs *WinfspFS) Rmdir(path string) int {
	fs.logger.Logf("[log] (Rmdir): path=%s", path)

	err := fs.client.Rmdir(path)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}
