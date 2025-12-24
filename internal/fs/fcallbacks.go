package fs

import (
	"os"
	"path"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/helpers"
)

func (fs *FuseFS) Truncate(p string, size int64, fh uint64) int {
	fs.logger.Logf("[Truncate]: path=%s fh=%d size=%d", p, fh, size)

	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	file, ok := fs.GetHandle(fh)
	if ok && file != nil && !file.Flags().WriteAllowed() {
		fs.logger.Errorf("[Truncate] access denied for %s, flag state: %+v return EACCES", p, file.Flags())
		return -EACCES
	}

	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Truncate] Path normalize error for path=%s error=%v return EIO", p, err)
		return -EIO
	}

	err = fs.client.Truncate(norm, size)
	if err != nil {
		fs.logger.Errorf("[Truncate] truncate error for path=%s size=%d: %v return EIO", p, size, err)
		return -EIO
	}

	return 0
}

func (fs *FuseFS) Unlink(p string) int {
	// Add handle deletion after successful removal
	fs.logger.Logf("[Unlink]: path=%s", p)
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	norm, err := casters.NormalizePath(p)
	if err != nil {
		fs.logger.Errorf("[Unlink] Path normalize error for path=%s error=%v return EIO", p, err)
		return -EIO
	}
	if err := fs.client.Remove(norm); err != nil {
		fs.logger.Errorf("[Unlink] remove error for path=%s: %v return EIO", p, err)
		return -EIO
	}

	return 0
}

func (fs *FuseFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	file, ok := fs.GetHandle(file_handle)
	if !ok {
		fs.logger.Errorf("[Write] invalid file handle=%d for path=%s returning EIO", file_handle, path)
		return -EIO
	}

	if !file.Flags().WriteAllowed() {
		fs.logger.Errorf("[Write] access denied for %s, flag state: %+v returning EACCES", path, file.Flags())
		return -EACCES
	}

	reqPageOffset, reqPageLen := helpers.PageAlignedRange(offset, int64(len(buffer)), file.remoteSize)
	if reqPageOffset != offset || reqPageLen != int64(len(buffer)) && !file.buffer.DirtyRange(reqPageOffset, reqPageLen) {
		fs.logger.Logf("[Write] adjusted write range for %s from offset=%d len=%d to offset=%d len=%d", path, offset, len(buffer), reqPageOffset, reqPageLen)
		remoteBuf, err := fs.client.ReadRange(path, reqPageOffset, reqPageLen)
		if err != nil && !helpers.IsNotExistErr(err) {
			if helpers.IsForbiddenErr(err) {
				fs.logger.Errorf("[Write] ReadRange forbidden for %s offset=%d len=%d: %v return EACCES", path, reqPageOffset, reqPageLen, err)
				return -EACCES
			}
			fs.logger.Errorf("[Write] ReadRange error for %s offset=%d len=%d: %v returning EIO", path, reqPageOffset, reqPageLen, err)
			return -EIO
		}
		if len(remoteBuf) > 0 {
			file.AddRemoteToBuffer(reqPageOffset, remoteBuf)
		}
	}

	file.AddToBuffer(offset, buffer)
	end := offset + int64(len(buffer))
	if end > file.stat.Size {
		file.stat.Size = end
	}

	buf := file.CopyBuffer()
	fs.logger.Logf("[Write] buffer len=%d offset=%d, after write: path=%s base=%d len=%d dirty=%v", len(buffer), offset, path, buf.Base, len(buf.Data), file.IsDirty())

	return len(buffer)
}

func (fs *FuseFS) Create(p string, flags int, mode uint32) (int, uint64) {
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	if err := fs.client.Create(p); err != nil {
		if os.IsPermission(err) || helpers.IsForbiddenErr(err) {
			fs.logger.Errorf("[Create]: permission denied for path=%s returning EACCES", p)
			return -EACCES, 0
		}
		fs.logger.Errorf("[Create]: remote write failed path=%s err=%v returning EIO", p, err)
		return -EIO, 0
	}

	// synthesize Stat_t immediately so Create is one RPC (PUT)
	isHidden := strings.HasPrefix(path.Base(p), ".")
	stat := casters.EmptyFileStat(isHidden)

	h := fs.NewHandle(p, stat, uint32(flags))

	go func(h uint64, p string) {
		fi, err := fs.client.Stat(p)
		if err != nil {
			return
		}
		fh, ok := fs.GetHandle(h)
		if !ok {
			return
		}
		fh.MLock()
		fh.stat = casters.FileInfoCast(fi)
		fh.remoteSize = fi.Size()
		fh.MUnlock()
	}(h, p)

	fs.logger.Logf("[Create] returning handle=%d path=%s flags=%#o mode=%#o", h, p, flags, mode)
	return 0, h
}

// Release should flush buffered segments (if any) to remote before closing.
func (fs *FuseFS) Release(path string, file_handle uint64) (errc int) {
	fs.logger.Logf("[Release] path=%s handle=%d", path, file_handle)
	defer fs.ReleaseHandle(file_handle)
	return 0
}

func (fs *FuseFS) Flush(path string, file_handle uint64) (errc int) {
	fh, ok := fs.GetHandle(file_handle)
	if !ok {
		fs.logger.Errorf("[Flush] invalid file handle=%d for path=%s", file_handle, path)
		return 0
	}

	if !fh.IsDirty() {
		return 0
	}

	if !fh.Flags().WriteAllowed() {
		return 0
	}

	fh.MLock()
	defer fh.MUnlock()
	buf := fh.CopyBuffer()
	fs.logger.Logf("[Flush] about to write path=%s buffer_len=%d buffer_off=%d", fh.Path(), len(buf.Data), buf.Base)
	if err := fs.client.WriteOffset(fh.Path(), buf.Data, buf.Base); err != nil {
		if helpers.IsForbiddenErr(err) {
			fs.logger.Logf("[Flush] WriteOffset forbidden for %s: %v; returning EACCES", fh.Path(), err)
			return -EACCES
		}
		if helpers.IsNotExistErr(err) && fh.Flags().Create() {
			end := buf.Base + int64(len(buf.Data))
			if end > int64(^uint(0)>>1) {
				fs.logger.Logf("[Flush] too large allocate for %s; returning EIO", fh.Path())
				return -EIO
			}
			full := make([]byte, int(end))
			copy(full[int(buf.Base):], buf.Data)
			if werr := fs.client.Write(fh.Path(), full); werr != nil {
				fs.logger.Logf("[Flush] Write error for %s: %v; returning EIO", fh.Path(), werr)
				return -EIO
			}
		} else {
			fs.logger.Logf("[Flush] WriteOffset error for %s: %v; returning EIO", fh.Path(), err)
			return -EIO
		}
	}
	fh.ClearBuffer()
	fh.remoteSize = buf.Base + int64(len(buf.Data))

	return 0
}

func (fs *FuseFS) Fsync(path string, datasync bool, file_handle uint64) (errc int) {
	fs.logger.Logf("[Fsync]: path=%s fh=%d datasync=%v", path, file_handle, datasync)
	return 0
}

func (fs *FuseFS) Access(path string, mode uint32) int {
	fs.logger.Logf("[Access]: path=%s mode=%#o", path, mode)
	norm, err := casters.NormalizePath(path)
	if err != nil {
		fs.logger.Errorf("[Access] Path normalize error for path=%s error=%v returning EIO", path, err)
		return -EIO
	}

	_, err = fs.client.Stat(norm)
	if err != nil {
		if helpers.IsNotExistErr(err) {
			fs.logger.Errorf("[Access] path=%s not found returning ENOENT", path)
			return -ENOENT
		} else if helpers.IsForbiddenErr(err) {
			fs.logger.Errorf("[Access] permission denied for path=%s returning EACCES", path)
			return -EACCES
		}
		fs.logger.Errorf("[Access] Stat error for path=%s: %v returning EIO", path, err)
		return -EIO
	}

	return 0
}

func (fs *FuseFS) Read(path string, buffer []byte, offset int64, file_handle uint64) int {
	fh, ok := fs.GetHandle(file_handle)
	if !ok {
		fs.logger.Errorf("[Read] invalid file handle=%d for path=%s returning EIO", file_handle, path)
		return -EIO
	}

	if !fh.Flags().ReadAllowed() {
		fs.logger.Errorf("[Read] access denied for %s, flag state: %+v returning EACCES", path, fh.Flags())
		return -EACCES
	}

	// 1. Check dirty buffer for overlapping pages
	// 2. Fetch remote state for file on buffer miss
	// 3. Merge with letting buffer override
	// 4. Should fill buffer with fetced info

	// temporarily do full read to test speed increase from buffering

	// requested window
	reqStart := offset
	reqLen := int64(len(buffer))
	if reqLen == 0 {
		return 0
	}

	buf := fh.CopyBuffer()

	if buf.Mask.IsDirtyRange(reqStart, reqLen) {
		fs.logger.Logf("[Read] dirty buffer full hit for %s offset=%d len=%d", path, reqStart, reqLen)
		goto merge
	}

	if reqStart <= fh.remoteSize {
		actualLen := min(reqLen, fh.remoteSize-reqStart)
		if actualLen <= 0 {
			goto merge
		}

		reqPageStart, reqPageLen := helpers.PageAlignedRange(reqStart, actualLen, fh.remoteSize)

		remoteBuf, err := fs.client.ReadRange(fh.Path(), reqPageStart, reqPageLen)
		if err == nil {
			if len(remoteBuf) > 0 {
				fs.logger.Logf("[Read] fetched remote data to fill buffer gap for %s offset=%d len=%d", path, reqPageStart, reqPageLen)
				fh.AddRemoteToBuffer(reqPageStart, remoteBuf)
			}
			goto merge
		}

		if helpers.IsNotExistErr(err) {
			goto merge
		} else if helpers.IsForbiddenErr(err) {
			fs.logger.Errorf("[Read] ReadRange forbidden for %s offset=%d len=%d: %v return EACCES", path, reqStart, reqLen, err)
			return -EACCES
		} else {
			fs.logger.Errorf("[Read] ReadRange error for %s offset=%d len=%d: %v return EIO", path, reqStart, reqLen, err)
			return -EIO
		}
	}

merge:
	dirtyBufRangeStart := max(reqStart-buf.Base, 0)
	dirtyBufRangeEnd := min(dirtyBufRangeStart+reqLen, int64(len(buf.Data)))
	n := copy(buffer, buf.Data[dirtyBufRangeStart:dirtyBufRangeEnd])

	return n
}
