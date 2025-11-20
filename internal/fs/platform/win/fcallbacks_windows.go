package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/helpers"
	"github.com/winfsp/cgofuse/fuse"
)

func (f *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	f.logger.Logf("[log] (Truncate): path=%s fh=%d size=%d", p, fh, size)

	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	file, ok := f.GetHandle(fh)
	if !ok {
		return -fuse.EIO
	}

	if !file.flags.WriteAllowed() {
		fmt.Println("Truncate forbidden")
		return -fuse.EACCES
	}

	err := f.client.Truncate(p, size)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Unlink(p string) int {
	f.logger.Logf("[log] (Unlink): path=%s", p)
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	if err := f.client.Remove(p); err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	f.logger.Logf("[log] (Write): path=%s fh=%d offset=%d len=%d", path, file_handle, offset, len(buffer))

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -fuse.EIO
	}

	if !file.flags.WriteAllowed() {
		fmt.Println("Write forbidden")
		return -fuse.EACCES
	}

	file.mu.Lock()
	defer file.mu.Unlock()

	if file.segments == nil {
		file.segments = make(map[int64][]byte)
	}

	// copy buffer to avoid referencing caller memory
	data := make([]byte, len(buffer))
	copy(data, buffer)
	file.segments[offset] = data
	file.dirty = true

	// update in-memory size if we extended beyond current known size
	end := offset + int64(len(buffer))
	if end > file.size {
		file.size = end
	}

	return len(buffer)
}

func (f *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	f.logger.Logf("[log] (Create): path=%s flags=%#o mode=%#o", path, flags, mode)

	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	if err := f.client.Create(path); err != nil {
		f.logger.Errorf("[log] (Create): remote write failed path=%s err=%v", path, err)
		if os.IsPermission(err) {
			return -fuse.EACCES, 0
		}
		return -fuse.EIO, 0
	}

	h := uint64(0)
	if fi, err := f.client.Stat(path); err == nil {
		h = f.NewHandle(path, casters.FileInfoCast(fi), uint32(flags))
	}

	f.logger.Logf("[log] (Create): returning handle=%d path=%s flags=%#o mode=%#o", h, path, flags, mode)
	return 0, h
}

// Release should flush buffered segments (if any) to remote before closing.
func (f *WinfspFS) Release(path string, file_handle uint64) (errc int) {
	f.logger.Logf("[Release] path=%s handle=%d", path, file_handle)

	fh, ok := f.GetHandle(file_handle)
	if !ok {
		goto cleanup
	}

	if !fh.flags.WriteAllowed() {
		goto cleanup
	}

	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.dirty && len(fh.segments) > 0 {
		base, err := f.client.Read(fh.path)
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
		fh.size = int64(len(merged))

		if err := f.client.Write(fh.path, merged); err != nil {
			f.logger.Errorf("[Release] write flush error=%v path=%s", err, fh.path)
			errc = -fuse.EIO
			goto cleanup
		}
	}

cleanup:
	fh.segments = nil
	fh.dirty = false
	f.handles.Delete(file_handle)
	return errc
}
