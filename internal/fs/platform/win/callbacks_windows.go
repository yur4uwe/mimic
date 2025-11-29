package win

import (
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/checks"
	"github.com/mimic/internal/core/flags"
	"github.com/winfsp/cgofuse/fuse"
)

const (
	DEFAULT_BLOCK_SIZE = 4096
	READ_LEN           = 1024 * 1024
)

type openedFile struct {
	path  string
	flags flags.OpenFlag
	size  int64
	stat  *fuse.Stat_t

	mu sync.Mutex
	// segments map[int64][]byte
	buffer []byte
	dirty  bool
}

func (of *openedFile) Lock() {
	of.mu.Lock()
}

func (of *openedFile) Unlock() {
	of.mu.Unlock()
}

func (fs *WinfspFS) NewHandle(path string, stat *fuse.Stat_t, oflags uint32) uint64 {
	file_handle := atomic.AddUint64(&fs.nextHandle, 1)
	fs.handles.Store(file_handle, &openedFile{
		path:  path,
		flags: flags.OpenFlag(oflags),
		size:  stat.Size,
		stat:  stat,
	})
	return file_handle
}

func (fs *WinfspFS) GetHandle(handle uint64) (*openedFile, bool) {
	file, ok := fs.handles.Load(handle)
	if !ok {
		return nil, false
	}
	of := file.(*openedFile)
	return of, true
}

func (fs *WinfspFS) Getattr(p string, stat *fuse.Stat_t, fh uint64) int {
	if p == "/" {
		stat.Mode = fuse.S_IFDIR | 0o777
		stat.Nlink = 2
		stat.Size = 0
		stat.Uid = 1000
		stat.Gid = 1000
		stat.Mtim = fuse.Now()
		stat.Atim = fuse.Now()
		stat.Ctim = fuse.Now()
		stat.Blksize = 4096
		stat.Birthtim = fuse.Now()
		return 0
	}

	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Getattr] Path normalize error for path=%s error=%v", p, err)
		return -fuse.ENOENT
	}

	if fi, ok := fs.GetHandle(fh); ^uint64(0) != fh && ok {
		*stat = *fi.stat
		stat.Size = fi.size

		fs.logger.Logf("[Getattr] found handle path=%s fh=%d mode=%#o size=%d", norm, fh, stat.Mode, stat.Size)
		return 0
	}

	file, err := fs.client.Stat(norm)
	if err != nil {
		return -fuse.ENOENT
	}

	*stat = *casters.FileInfoCast(file)

	fs.logger.Logf("[Getattr] path=%s has fh=%t mode=%#o size=%d", norm, fh^(^uint64(0)) == 0, file.Mode(), file.Size())

	return 0
}

func (fs *WinfspFS) Open(path string, oflags int) (int, uint64) {
	flags := flags.OpenFlag(uint32(oflags))

	fi, err := fs.client.Stat(path)

	if checks.IsNilInterface(fi) {
		err = os.ErrNotExist
	}

	if flags.Create() && os.IsNotExist(err) {
		if err := fs.client.Create(path); err != nil {
			return -fuse.EIO, 0
		}
	}

	if flags.Exclusive() && !os.IsNotExist(err) {
		return -fuse.EEXIST, 0
	}

	handle := fs.NewHandle(path, casters.FileInfoCast(fi), uint32(flags))

	fs.logger.Logf("[Open] path=%s flags=%d handle=%d", path, flags, handle)

	return 0, handle
}

func (fs *WinfspFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {
	fs.logger.Logf("[Read] path=%s offset=%d len=%d fh=%d", path, offset, len(buffer), file_handle)

	file, ok := fs.GetHandle(file_handle)
	if !ok {
		return -fuse.EIO
	}

	if !file.flags.ReadAllowed() {
		return -fuse.EACCES
	}

	if offset >= file.size {
		return 0 // EOF
	}

	toRead := len(buffer)
	rc, err := fs.client.ReadRange(file.path, offset, int64(toRead))
	if err != nil {
		return -fuse.EIO
	}
	defer rc.Close()

	n, err := io.ReadFull(rc, buffer)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return n
	} else if err != nil {
		return -fuse.EIO
	}

	return n
}

func (fs *WinfspFS) Rename(oldPath string, newPath string) int {
	fs.logger.Logf("[Rename] from=%s to=%s", oldPath, newPath)

	err := fs.client.Rename(oldPath, newPath)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (fs *WinfspFS) Utimens(path string, times []fuse.Timespec) int {
	fs.logger.Logf("[Utimens] path=%s times=%#v", path, times)
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// no direct support for setting times in WebDAV; ignore for now
	return 0
}

func (fs *WinfspFS) Statfs(path string, stat *fuse.Statfs_t) int {
	fs.logger.Logf("[Statfs] path=%s", path)

	// provide some reasonable defaults
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
