package win

import (
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mimic/internal/core/casters"
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

	mu       sync.Mutex
	segments map[int64][]byte
	dirty    bool
}

func (f *WinfspFS) NewHandle(path string, stat *fuse.Stat_t, oflags uint32) uint64 {
	file_handle := atomic.AddUint64(&f.nextHandle, 1)
	f.handles.Store(file_handle, &openedFile{
		path:  path,
		flags: flags.OpenFlag(oflags),
		size:  stat.Size,
		stat:  stat,
	})
	return file_handle
}

func (f *WinfspFS) GetHandle(handle uint64) (*openedFile, bool) {
	file, ok := f.handles.Load(handle)
	if !ok {
		return nil, false
	}
	of := file.(*openedFile)
	return of, true
}

func (f *WinfspFS) Getattr(p string, stat *fuse.Stat_t, fh uint64) int {
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
		f.logger.Errorf("[Getattr] Path normalize error for path=%s error=%v", p, err)
		return -fuse.ENOENT
	}

	if fi, ok := f.GetHandle(fh); ^uint64(0) != fh && ok {
		*stat = *fi.stat

		f.logger.Logf("[Getattr] found handle path=%s fh=%d mode=%#o size=%d", norm, fh, stat.Mode, stat.Size)
		return 0
	}

	file, err := f.client.Stat(norm)
	if err != nil {
		return -fuse.ENOENT
	}

	*stat = *casters.FileInfoCast(file)

	f.logger.Logf("[Getattr] path=%s has fh=%t mode=%#o size=%d", norm, fh^(^uint64(0)) == 0, stat.Mode, stat.Size)

	return 0
}

func (f *WinfspFS) Open(path string, flags int) (int, uint64) {
	fi, err := f.client.Stat(path)
	if err != nil {
		return -fuse.EIO, 0
	}

	if fi.IsDir() {
		return -fuse.EISDIR, 0
	}

	handle := f.NewHandle(path, casters.FileInfoCast(fi), uint32(flags))

	f.logger.Logf("[Open] path=%s flags=%d handle=%d", path, flags, handle)

	return 0, handle
}

func (f *WinfspFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {
	f.logger.Logf("[Read] path=%s offset=%d fh=%d", path, offset, file_handle)

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -fuse.EIO
	}

	if offset >= file.size {
		return 0 // EOF
	}

	toRead := len(buffer)
	rc, err := f.client.ReadRange(file.path, offset, int64(toRead))
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

func (f *WinfspFS) Rename(oldPath string, newPath string) int {
	f.logger.Logf("[Rename] from=%s to=%s", oldPath, newPath)

	err := f.client.Rename(oldPath, newPath)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Utimens(path string, times []fuse.Timespec) int {
	f.logger.Logf("[Utimens] path=%s times=%#v", path, times)
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// no direct support for setting times in WebDAV; ignore for now
	return 0
}
