//go:build linux

package casters

import (
	"os"

	"bazil.org/fuse"
)

func FileInfoCast(f os.FileInfo) *fuse.Attr {
	attr := &fuse.Attr{
		Mode:  f.Mode(),         // base mode
		Size:  uint64(f.Size()), // size in bytes
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
		// regular file defaults
		attr.Mode |= 0644
		attr.Nlink = 1
		// compute blocks using 4096 block size
		if f.Size() > 0 {
			attr.Blocks = uint64((f.Size() + 4095) / 4096)
		}
	}

	return attr
}
