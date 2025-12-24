package casters

import (
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/winfsp/cgofuse/fuse"
)

func NormalizePath(p string) (string, error) {
	s, err := url.PathUnescape(p)
	if err != nil {
		return "", err
	}

	s = strings.ReplaceAll(s, "\\", "/")
	s = path.Clean(s)
	return s, nil
}

func EmptyFileStat(hidden bool) *fuse.Stat_t {
	uid, gid, _ := fuse.Getcontext()
	Flags := uint32(0)
	if hidden {
		Flags = fuse.UF_HIDDEN
	}
	return &fuse.Stat_t{
		Mode:     fuse.S_IFREG | 0o644,
		Nlink:    1,
		Size:     0,
		Uid:      uid,
		Gid:      gid,
		Atim:     fuse.NewTimespec(time.Now()),
		Mtim:     fuse.NewTimespec(time.Now()),
		Ctim:     fuse.NewTimespec(time.Now()),
		Blksize:  4096,
		Birthtim: fuse.NewTimespec(time.Now()),
		Flags:    Flags,
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

	uid, gid, _ := fuse.Getcontext()

	stat.Uid = uid
	stat.Gid = gid
	stat.Mtim = fuse.NewTimespec(f.ModTime())
	stat.Atim = fuse.NewTimespec(f.ModTime())
	stat.Ctim = fuse.NewTimespec(f.ModTime())
	stat.Blksize = 4096
	stat.Birthtim = fuse.NewTimespec(f.ModTime())

	if strings.HasPrefix(f.Name(), ".") {
		stat.Flags = fuse.UF_HIDDEN
	}

	return stat
}
