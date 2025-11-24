package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/helpers"
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

	if !file.flags.WriteAllowed() {
		fmt.Println("Truncate forbidden")
		return -fuse.EACCES
	}

	err := fs.client.Truncate(p, size)
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
	if err := fs.client.Remove(p); err != nil {
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

	if !file.flags.WriteAllowed() {
		fmt.Println("Write forbidden")
		return -fuse.EACCES
	}

	file.Lock()
	defer file.Unlock()
	if err := fs.client.WriteOffset(path, buffer, offset); err != nil {
		return -fuse.EIO
	}

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

	fh, ok := fs.GetHandle(file_handle)
	if !ok {
		goto cleanup
	}

	if !fh.flags.WriteAllowed() {
		goto cleanup
	}

	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.dirty && len(fh.segments) > 0 {
		base, err := fs.client.Read(fh.path)
		if err != nil {
			// if not exist, treat base as empty
			if os.IsNotExist(err) {
				base = []byte{}
			} else {
				errc = -fuse.EIO
				goto cleanup
			}
		}

		merged := helpers.MergeSegmentsInto(base, fh.segments)
		fmt.Println("Merged Length:", len(merged))

		if err := fs.client.Write(fh.path, merged); err != nil {
			fs.logger.Errorf("[Release] write flush error=%v path=%s", err, fh.path)
			errc = -fuse.EIO
			goto cleanup
		}

		if fi, err := fs.client.Stat(fh.path); err == nil {
			if int64(len(merged)) != fi.Size() {
				fs.logger.Errorf("[Release] size mismatch after write path=%s want=%d got=%d", fh.path, len(merged), fi.Size())
				errc = -fuse.EIO
				goto cleanup
			}
		}
	}

cleanup:
	fh.segments = nil
	fh.dirty = false
	fs.handles.Delete(file_handle)
	return errc
}
