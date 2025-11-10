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
	perm := uint32(f.Mode().Perm())

	if f.IsDir() {
		stat.Mode = fuse.S_IFDIR | perm
		stat.Nlink = 2
		stat.Size = 0
	} else {
		stat.Mode = fuse.S_IFREG | perm
		stat.Nlink = 1
		stat.Size = f.Size()
	}

	// give a non-zero inode value (simple stable-ish value)
	uid, gid, _ := fuse.Getcontext()

	stat.Uid = uid
	stat.Gid = gid
	stat.Mtim = fuse.NewTimespec(f.ModTime())
	stat.Atim = fuse.NewTimespec(f.ModTime())
	stat.Ctim = fuse.NewTimespec(f.ModTime())
	stat.Blksize = 4096
	stat.Birthtim = fuse.NewTimespec(f.ModTime())

	stat.Flags = fuse.UF_ARCHIVE

	return stat
}
