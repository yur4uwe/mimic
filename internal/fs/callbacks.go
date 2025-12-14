package fs

import (
	"os"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/checks"
	"github.com/mimic/internal/core/flags"
	"github.com/mimic/internal/core/helpers"
	fuselib "github.com/winfsp/cgofuse/fuse"
)

const (
	DEFAULT_BLOCK_SIZE = 4096
	READ_LEN           = 1024 * 1024
)

func (fs *FuseFS) Getattr(p string, stat *fuselib.Stat_t, fh uint64) int {
	if p == "/" {
		stat.Mode = fuselib.S_IFDIR | 0o777
		stat.Nlink = 2
		stat.Size = 0
		stat.Uid = 1000
		stat.Gid = 1000
		stat.Mtim = fuselib.Now()
		stat.Atim = fuselib.Now()
		stat.Ctim = fuselib.Now()
		stat.Blksize = 4096
		stat.Birthtim = fuselib.Now()
		return 0
	}

	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Getattr] Path normalize error for path=%s error=%v", p, err)
		return -ENOENT
	}

	if fi, ok := fs.GetHandle(fh); ^uint64(0) != fh && ok {
		*stat = *fi.stat

		if fi.buffer != nil {
			bufSize := fi.buffer.Size()
			bufBase := fi.buffer.BasePos()
			if bufBase+bufSize > stat.Size {
				stat.Size = bufBase + bufSize
			}
		}

		fs.logger.Logf("[Getattr] found handle path=%s fh=%d mode=%#o size=%d", norm, fh, stat.Mode, stat.Size)
		return 0
	}

	file, err := fs.client.Stat(norm)
	if err != nil {
		if helpers.IsNotExistErr(err) {
			fs.logger.Errorf("[Getattr] stat: %s not found: %v; returning ENOENT", norm, err)
			return -ENOENT
		}
		fs.logger.Errorf("[Getattr] stat error for %s: %v; returning EIO", norm, err)
		return -EIO
	}

	if checks.IsNilInterface(file) {
		fs.logger.Errorf("[Getattr] nil fileinfo for %s; returning ENOENT", norm)
		return -ENOENT
	}

	*stat = *casters.FileInfoCast(file)

	buf, ok := fs.bufferCache.Get(norm)
	if ok {
		bufSize := buf.Size()
		bufBase := buf.BasePos()
		stat.Size = max(stat.Size, bufBase+bufSize)
	}

	fs.logger.Logf("[Getattr] path=%s has fh=%t mode=%#o size=%d", norm, fh^(^uint64(0)) == 0, file.Mode(), file.Size())

	return 0
}

func (fs *FuseFS) Open(path string, oflags int) (int, uint64) {
	flags := flags.OpenFlag(uint32(oflags))

	fi, err := fs.client.Stat(path)

	if checks.IsNilInterface(fi) {
		err = os.ErrNotExist
	}

	if flags.Create() && helpers.IsNotExistErr(err) {
		if err := fs.client.Create(path); err != nil {
			fs.logger.Errorf("[Open] remote create failed path=%s err=%v", path, err)
			return -EIO, 0
		}
	}

	if flags.Exclusive() && !helpers.IsNotExistErr(err) {
		fs.logger.Errorf("[Open] file exists and exclusive flag set path=%s", path)
		return -EEXIST, 0
	}

	handle := fs.NewHandle(path, casters.FileInfoCast(fi), uint32(flags))

	fs.logger.Logf("[Open] path=%s flags=%d handle=%d", path, flags, handle)

	return 0, handle
}

func (fs *FuseFS) Rename(oldPath string, newPath string) int {
	fs.logger.Logf("[Rename] from=%s to=%s", oldPath, newPath)

	err := fs.client.Rename(oldPath, newPath)
	if err != nil {
		fs.logger.Errorf("[Rename] rename error from %s to %s: %v", oldPath, newPath, err)
		return -EIO
	}

	return 0
}

func (fs *FuseFS) Utimens(path string, times []fuselib.Timespec) int {
	fs.logger.Logf("[Utimens] path=%s times=%#v", path, times)
	// no direct support for setting times in WebDAV; ignore for now
	return 0
}

func (fs *FuseFS) Statfs(path string, stat *fuselib.Statfs_t) int {
	fs.logger.Logf("[Statfs] path=%s", path)

	stat.Bsize = DEFAULT_BLOCK_SIZE
	stat.Frsize = DEFAULT_BLOCK_SIZE
	stat.Blocks = 1024 * 1024 // 1M blocks
	stat.Bfree = 512 * 1024   // 50% free
	stat.Bavail = 512 * 1024  // 50% free
	stat.Files = 1024 * 1024
	stat.Ffree = 512 * 1024
	stat.Favail = 512 * 1024
	stat.Namemax = 255

	return 0
}

func (fs *FuseFS) Chmod(path string, mode uint32) int {
	fs.logger.Logf("[Chmod] path=%s mode=%#o", path, mode)
	return -ENOSYS
}

func (fs *FuseFS) Chown(path string, uid uint32, gid uint32) int {
	fs.logger.Logf("[Chown] path=%s uid=%d gid=%d", path, uid, gid)
	return -ENOSYS
}

func (fs *FuseFS) Destroy() {
	fs.logger.Logf("[Destroy] called")
}

func (fs *FuseFS) Fsyncdir(path string, datasync bool, fh uint64) int {
	fs.logger.Logf("[Fsyncdir] path=%s datasync=%v fh=%d", path, datasync, fh)
	return -ENOSYS
}

func (fs *FuseFS) Getxattr(path string, name string) (int, []byte) {
	fs.logger.Logf("[Getxattr] path=%s name=%s", path, name)
	return -ENOSYS, nil
}

func (fs *FuseFS) Init() {
	fs.logger.Logf("[Init] called")
}

func (fs *FuseFS) Link(oldpath string, newpath string) int {
	fs.logger.Logf("[Link] oldpath=%s newpath=%s", oldpath, newpath)
	return -ENOSYS
}

func (fs *FuseFS) Listxattr(path string, fill func(name string) bool) int {
	fs.logger.Logf("[Listxattr] path=%s", path)
	return -ENOSYS
}

func (fs *FuseFS) Mknod(path string, mode uint32, dev uint64) int {
	fs.logger.Logf("[Mknod] path=%s mode=%#o dev=%d", path, mode, dev)
	return -ENOSYS
}

func (fs *FuseFS) Readlink(path string) (int, string) {
	fs.logger.Logf("[Readlink] path=%s", path)
	return -ENOSYS, ""
}

func (fs *FuseFS) Removexattr(path string, name string) int {
	fs.logger.Logf("[Removexattr] path=%s name=%s", path, name)
	return -ENOSYS
}

func (fs *FuseFS) Setxattr(path string, name string, value []byte, flags int) int {
	fs.logger.Logf("[Setxattr] path=%s name=%s flags=%d", path, name, flags)
	return -ENOSYS
}

func (fs *FuseFS) Symlink(target string, newpath string) int {
	fs.logger.Logf("[Symlink] target=%s newpath=%s", target, newpath)
	return -ENOSYS
}
