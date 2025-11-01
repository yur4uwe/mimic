//go:build linux

package casters

import (
	"os"

	"bazil.org/fuse"
)

func FileInfoCast(f os.FileInfo) *fuse.Attr {
	attr := &fuse.Attr{
		Mode:  f.Mode(),
		Size:  uint64(f.Size()),
		Uid:   uint32(0),
		Gid:   uint32(0),
		Atime: f.ModTime(),
		Mtime: f.ModTime(),
		Ctime: f.ModTime(),
	}

	if f.IsDir() {
		attr.Mode |= os.ModeDir | 0755
		attr.Nlink = 2
		attr.Size = 0
	} else {
		attr.Mode |= 0644
		attr.Nlink = 1
		if f.Size() > 0 {
			attr.Blocks = uint64((f.Size() + 4095) / 4096)
		}
	}

	return attr
}
