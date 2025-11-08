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
		stat.Mode = fuse.S_IFDIR | 00777
		stat.Nlink = 2
		stat.Size = 0
	} else {
		stat.Mode = fuse.S_IFREG | 00777
		stat.Nlink = 1
		stat.Size = f.Size()
	}

	// give a non-zero inode value (simple stable-ish value)
	stat.Ino = uint64(f.ModTime().UnixNano())

	stat.Uid = 1000
	stat.Gid = 1000
	stat.Mtim = fuse.NewTimespec(f.ModTime())
	stat.Atim = fuse.NewTimespec(f.ModTime())
	stat.Ctim = fuse.NewTimespec(f.ModTime())
	stat.Blksize = 4096
	// blocks in 512-byte units
	if stat.Size > 0 {
		stat.Blocks = (stat.Size + 511) / 512
	} else {
		stat.Blocks = 0
	}
	stat.Birthtim = fuse.NewTimespec(f.ModTime())

	stat.Flags = 0

	return stat
}
