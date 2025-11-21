package casters

import (
	"os"
	"syscall"
	"time"

	"bazil.org/fuse"
)

func FileInfoCast(f os.FileInfo) *fuse.Attr {
	attr := &fuse.Attr{
		Valid: time.Second,
		Mode:  f.Mode(),
		Size:  uint64(f.Size()),
		Uid:   uint32(os.Getuid()),
		Gid:   uint32(os.Getgid()),
		Atime: f.ModTime(),
		Mtime: f.ModTime(),
		Ctime: f.ModTime(),
	}

	if st, ok := f.Sys().(*syscall.Stat_t); ok {
		attr.Uid = uint32(st.Uid)
		attr.Gid = uint32(st.Gid)
	}

	if f.IsDir() {
		attr.Mode |= os.FileMode(0o755)
		attr.Nlink = 2
		attr.Size = 0
	} else {
		attr.Mode |= os.FileMode(0o666)
		attr.Nlink = 1
	}

	return attr
}
