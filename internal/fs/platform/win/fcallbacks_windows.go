package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (f *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	fmt.Printf("[log] (Truncate): path=%s fh=%d size=%d\n", p, fh, size)

	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	err := f.client.Truncate(p, size)
	if err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Unlink(p string) int {
	fmt.Printf("[log] (Unlink): path=%s\n", p)
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	if err := f.client.Remove(p); err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Printf("[log] (Write): path=%s fh=%d offset=%d len=%d - writing direct\n", path, file_handle, offset, len(buffer))

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -fuse.EIO
	}

	file.mu.Lock()
	defer file.mu.Unlock()

	if err := f.client.WriteOffset(file.path, buffer, offset); err != nil {
		fmt.Printf("[log] (Write): remote write error=%v path=%s fh=%d offset=%d len=%d\n", err, file.path, file_handle, offset, len(buffer))
		return -fuse.EIO
	}

	end := offset + int64(len(buffer))
	if end > file.size {
		file.size = end
	}

	return len(buffer)
}

func (f *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fmt.Printf("[log] (Create): path=%s flags=%#o mode=%#o\n", path, flags, mode)

	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	if err := f.client.Create(path); err != nil {
		fmt.Printf("[log] (Create): remote write failed path=%s err=%v\n", path, err)
		if os.IsPermission(err) {
			return -fuse.EACCES, 0
		}
		return -fuse.EIO, 0
	}

	h := uint64(0)
	if fi, err := f.client.Stat(path); err == nil {
		h = f.NewHandle(path, casters.FileInfoCast(fi))
	}

	fmt.Printf("[log] (Create): returning handle=%d path=%s flags=%#o mode=%#o\n", h, path, flags, mode)
	return 0, h
}
