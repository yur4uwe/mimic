package fs

import (
	"io"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/helpers"
)

func (fs *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	fs.logger.Logf("[Truncate]: path=%s fh=%d size=%d", p, fh, size)

	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	file, ok := fs.GetHandle(fh)
	if ok && file != nil && !file.Flags().WriteAllowed() {
		fs.logger.Errorf("[Truncate] access denied for %s, flag state: %+v", p, file.Flags())
		return -EACCES
	}

	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Truncate] Path normalize error for path=%s error=%v", p, err)
		return -EIO
	}

	err = fs.client.Truncate(norm, size)
	if err != nil {
		fs.logger.Errorf("[Truncate] truncate error for path=%s size=%d: %v", p, size, err)
		return -EIO
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
		return -EIO
	}
	if err := fs.client.Remove(norm); err != nil {
		fs.logger.Errorf("[Unlink] remove error for path=%s: %v", p, err)
		return -EIO
	}

	return 0
}

func (fs *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {

	file, ok := fs.GetHandle(file_handle)
	if !ok {
		fs.logger.Errorf("[Write] invalid file handle=%d for path=%s", file_handle, path)
		return -EIO
	}

	fs.logger.Logf("[Write]: path=%s fh=%d offset=%d len=%d flags=%s", path, file_handle, offset, len(buffer), file.Flags())
	if !file.Flags().WriteAllowed() {
		fs.logger.Errorf("[Write] access denied for %s, flag state: %+v", path, file.Flags())
		return -EACCES
	}

	file.MLock()
	file.AddToBuffer(offset, buffer)
	file.MUnlock()

	return len(buffer)
}

func (fs *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fs.logger.Logf("[Create]: path=%s flags=%#o mode=%#o", path, flags, mode)

	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	if err := fs.client.Create(path); err != nil {
		if os.IsPermission(err) {
			fs.logger.Errorf("[Create]: permission denied for path=%s", path)
			return -EACCES, 0
		}
		fs.logger.Errorf("[Create]: remote write failed path=%s err=%v", path, err)
		return -EIO, 0
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
		fs.logger.Errorf("[Flush] invalid file handle=%d for path=%s", file_handle, path)
		return 0
	}

	if !fh.IsDirty() {
		fs.logger.Logf("[Flush] no dirty data for path=%s", path)
		return 0
	}

	if !fh.Flags().WriteAllowed() {
		fs.logger.Errorf("[Flush] access denied for %s, flag state: %+v", path, fh.Flags())
		return 0
	}

	fh.MLock()
	defer fh.MUnlock()
	buf, off := fh.Buffer()
	fs.logger.Logf("[Flush] about to write path=%s buffer_len=%d buffer_off=%d", fh.Path(), len(buf), off)
	if err := fs.client.WriteOffset(fh.Path(), buf, off); err != nil {
		if helpers.IsNotExistErr(err) && fh.Flags().Create() {
			end := off + int64(len(buf))
			if end > int64(^uint(0)>>1) {
				fs.logger.Logf("[Flush] too large allocate for %s; returning EIO", fh.Path())
				return -EIO
			}
			full := make([]byte, int(end))
			copy(full[int(off):], buf)
			if werr := fs.client.Write(fh.Path(), full); werr != nil {
				fs.logger.Logf("[Flush] client.Write error for %s: %v; returning EIO", fh.Path(), werr)
				return -EIO
			}
		} else {
			fs.logger.Logf("[Flush] client.WriteOffset error for %s: %v; returning EIO", fh.Path(), err)
			return -EIO
		}
	}
	fh.ClearBuffer()

	return 0
}

func (fs *WinfspFS) Fsync(path string, datasync bool, file_handle uint64) (errc int) {
	fs.logger.Logf("[Fsync]: path=%s fh=%d datasync=%v", path, file_handle, datasync)
	return 0
}

func (fs *WinfspFS) Access(path string, mode uint32) int {
	fs.logger.Logf("[Access]: path=%s mode=%#o", path, mode)
	norm, err := casters.NormalizePath(path)
	if err != nil {
		fs.logger.Errorf("[Access] Path normalize error for path=%s error=%v", path, err)
		return -EIO
	}

	_, err = fs.client.Stat(norm)
	if err != nil {
		if helpers.IsNotExistErr(err) {
			return -ENOENT
		}
		fs.logger.Errorf("[Access] Stat error for path=%s: %v", path, err)
		return -EIO
	}

	return 0
}

func (fs *WinfspFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {

	fh, ok := fs.GetHandle(file_handle)
	if !ok {
		fs.logger.Errorf("[Read] invalid file handle=%d for path=%s", file_handle, path)
		return -EIO
	}

	if !fh.Flags().ReadAllowed() {
		fs.logger.Errorf("[Read] access denied for %s, flag state: %+v", path, fh.Flags())
		return -EACCES
	}

	// If no dirty buffer, keep previous simple path
	fs.logger.Logf("[Read] no dirty buffer path=%s offset=%d len=%d fh=%d", path, offset, len(buffer), file_handle)

	if offset >= fh.stat.Size {
		return 0
	}
	toRead := len(buffer)
	rc, err := fs.client.ReadRange(fh.Path(), offset, int64(toRead))
	if err != nil {
		fs.logger.Errorf("[Read] ReadRange error for %s offset=%d len=%d: %v", path, offset, toRead, err)
		return -EIO
	}
	defer rc.Close()

	n, err := io.ReadFull(rc, buffer)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return n
	} else if err != nil {
		fs.logger.Errorf("[Read] ReadFull error for %s offset=%d len=%d: %v", path, offset, toRead, err)
		return -EIO
	}
	return n
}
