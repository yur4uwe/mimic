package casters

import (
	"os"
	"time"

	"bazil.org/fuse"
)

func FileInfoCast(f os.FileInfo) *fuse.Attr {
	attr := &fuse.Attr{
		Valid: time.Minute,
		Mode:  f.Mode(),
		Size:  uint64(f.Size()),
		Uid:   uint32(1000),
		Gid:   uint32(1000),
		Atime: f.ModTime(),
		Mtime: f.ModTime(),
		Ctime: f.ModTime(),
	}

	if f.IsDir() {
		attr.Mode = os.ModeDir | 0o755
		attr.Nlink = 2
		attr.Size = 0
	} else {
		attr.Mode = os.FileMode(0o644)
		attr.Nlink = 1
		if f.Size() > 0 {
			attr.Blocks = uint64((f.Size() + 4095) / 4096)
		}
	}

	return attr
}
