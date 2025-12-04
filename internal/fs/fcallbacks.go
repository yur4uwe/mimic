package fs

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (fs *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	fs.logger.Logf("[Truncate]: path=%s fh=%d size=%d", p, fh, size)

	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	file, ok := fs.GetHandle(fh)
	if !ok {
		return -fuse.EIO
	}

	if !file.Flags().WriteAllowed() {
		fmt.Println("Truncate forbidden")
		return -fuse.EACCES
	}

	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Truncate] Path normalize error for path=%s error=%v", p, err)
		return -fuse.EIO
	}

	err = fs.client.Truncate(norm, size)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (fs *WinfspFS) Unlink(p string) int {
	fs.logger.Logf("[Unlink]: path=%s", p)
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Unlink] Path normalize error for path=%s error=%v", p, err)
		return -fuse.EIO
	}
	if err := fs.client.Remove(norm); err != nil {
		return -fuse.EIO
	}

	return 0
}

func (fs *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	fs.logger.Logf("[Write]: path=%s fh=%d offset=%d len=%d", path, file_handle, offset, len(buffer))

	file, ok := fs.GetHandle(file_handle)
	if !ok {
		return -fuse.EIO
	}

	if !file.Flags().WriteAllowed() {
		fmt.Println("Write forbidden")
		return -fuse.EACCES
	}

	file.MLock()
	// Add data into the handle buffer (will mark dirty via non-nil buffer)
	file.AddToBuffer(offset, buffer)
	file.MUnlock()

	return len(buffer)
}

func (fs *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fs.logger.Logf("[log] (Create): path=%s flags=%#o mode=%#o", path, flags, mode)

	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	if err := fs.client.Create(path); err != nil {
		fs.logger.Errorf("[Create]: remote write failed path=%s err=%v", path, err)
		if os.IsPermission(err) {
			return -fuse.EACCES, 0
		}
		return -fuse.EIO, 0
	}

	h := uint64(0)
	if fi, err := fs.client.Stat(path); err == nil {
		h = fs.NewHandle(path, casters.FileInfoCast(fi), uint32(flags))
	}

	fs.logger.Logf("[Create]: returning handle=%d path=%s flags=%#o mode=%#o", h, path, flags, mode)
	return 0, h
}

// Release should flush buffered segments (if any) to remote before closing.
func (fs *WinfspFS) Release(path string, file_handle uint64) (errc int) {
	fs.logger.Logf("[Release] path=%s handle=%d", path, file_handle)
	defer fs.handles.Delete(file_handle)
	return fs.Flush(path, file_handle)
}

func (fs *WinfspFS) Flush(path string, file_handle uint64) (errc int) {
	fs.logger.Logf("[Flush]: path=%s fh=%d", path, file_handle)
	fh, ok := fs.GetHandle(file_handle)
	if !ok {
		return 0
	}

	if !fh.Flags().WriteAllowed() {
		return 0
	}

	fh.MLock()
	defer fh.MUnlock()
	if fh.IsDirty() {
		buf, off := fh.Buffer()
		fs.logger.Logf("[Flush] about to write path=%s buffer_len=%d buffer_off=%d", fh.Path(), len(buf), off)
		if err := fs.client.WriteOffset(fh.Path(), buf, off); err != nil {
			if os.IsNotExist(err) && fh.Flags().Create() {
				end := off + int64(len(buf))
				if end > int64(^uint(0)>>1) {
					fs.logger.Logf("[Flush] too large allocate for %s; returning EIO", fh.Path())
					return -fuse.EIO
				}
				full := make([]byte, int(end))
				copy(full[int(off):], buf)
				if werr := fs.client.Write(fh.Path(), full); werr != nil {
					fs.logger.Logf("[Flush] client.Write error for %s: %v; returning EIO", fh.Path(), werr)
					return -fuse.EIO
				}
			} else {
				fs.logger.Logf("[Flush] client.WriteOffset error for %s: %v; returning EIO", fh.Path(), err)
				return -fuse.EIO
			}
		}
		fh.ClearBuffer()
	}

	return 0
}

func (fs *WinfspFS) Fsync(path string, datasync bool, file_handle uint64) (errc int) {
	fs.logger.Logf("[Fsync]: path=%s fh=%d datasync=%v", path, file_handle, datasync)
	return 0
}
