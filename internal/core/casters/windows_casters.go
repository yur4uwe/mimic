//go:build windows

package casters

import (
	"os"
	"syscall"

	"github.com/winfsp/cgofuse/fuse"
)

func TimeCast(t syscall.Filetime) fuse.Timespec {
	nsec := t.Nanoseconds()
	return fuse.Timespec{
		Sec:  nsec / 1e9,
		Nsec: nsec % 1e9,
	}
}

func FileInfoCast(f os.FileInfo) *fuse.Stat_t {
	stat := &fuse.Stat_t{}

	if f.IsDir() {
		stat.Mode = fuse.S_IFDIR | 00755
		stat.Nlink = 2
		stat.Size = 0
	} else {
		stat.Mode = fuse.S_IFREG | 00644
		stat.Nlink = 1
		stat.Size = f.Size()
	}

	stat.Uid = 1000
	stat.Gid = 1000
	stat.Mtim = fuse.NewTimespec(f.ModTime())
	stat.Atim = fuse.NewTimespec(f.ModTime())
	stat.Ctim = fuse.NewTimespec(f.ModTime())
	stat.Blksize = 4096
	stat.Birthtim = fuse.NewTimespec(f.ModTime())

	return stat
}
