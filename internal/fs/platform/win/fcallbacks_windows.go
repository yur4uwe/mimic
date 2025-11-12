package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/casters"
	"github.com/winfsp/cgofuse/fuse"
)

func (f *WinfspFS) Mknod(path string, mode uint32, dev uint64) int {
	fmt.Printf("[log] (Mknod): path=%s mode=%#o dev=%d\n", path, mode, dev)

	// Only support regular file creation via mknod; other node types are unsupported.
	if mode&fuse.S_IFMT != fuse.S_IFREG {
		fmt.Printf("[log] (Mknod): only regular files supported path=%s mode=%#o\n", path, mode)
		return -fuse.ENOSYS
	}

	// normalize path
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// create an empty file on remote (PUT)
	if err := f.client.Create(path); err != nil {
		fmt.Printf("[log] (Mknod): remote write failed path=%s err=%v\n", path, err)
		return -fuse.EIO
	}

	// success
	return 0
}

func (f *WinfspFS) Readlink(path string) (int, string) {
	fmt.Printf("[log] (Readlink): path=%s\n", path)
	// return symlink target
	return -fuse.ENOSYS, ""
}

func (f *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	fmt.Printf("[log] (Truncate): path=%s fh=%d size=%d\n", p, fh, size)
	// implement truncate by reading whole file, resizing, and PUTting back.
	// This is inefficient but works with servers that only support full PUT.
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	data, err := f.client.Read(p)
	if err != nil {
		// if file doesn't exist, report fuse.ENOENT
		if os.IsNotExist(err) {
			return -fuse.ENOENT
		}
		return -fuse.EIO
	}

	var newdata []byte
	if int64(len(data)) > size {
		newdata = data[:size]
	} else if int64(len(data)) == size {
		return 0 // nothing to do
	} else {
		// extend with zeros
		newdata = make([]byte, size)
		copy(newdata, data)
	}

	if err := f.client.Write(p, newdata); err != nil {
		return -fuse.EIO
	}

	return 0
}

func (f *WinfspFS) Unlink(p string) int {
	fmt.Printf("[log] (Unlink): path=%s\n", p)
	// delete file
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

	// Write the provided buffer directly to the remote backend at the
	// requested offset using a ranged stream write.
	if err := f.client.WriteOffset(file.path, buffer, offset); err != nil {
		fmt.Printf("[log] (Write): remote write error=%v path=%s fh=%d offset=%d len=%d\n", err, file.path, file_handle, offset, len(buffer))
		return -fuse.EIO
	}

	// update cached size
	end := offset + int64(len(buffer))
	if end > file.size {
		file.size = end
	}

	return len(buffer)
}

func (f *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fmt.Printf("[log] (Create): path=%s flags=%#o mode=%#o\n", path, flags, mode)

	// normalize
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// create empty remote file (PUT). Use wrapper which updates cache.
	if err := f.client.Write(path, []byte{}); err != nil {
		fmt.Printf("[log] (Create): remote write failed path=%s err=%v\n", path, err)
		// map common remote permission errors where possible
		if os.IsPermission(err) {
			return -fuse.EACCES, 0
		}
		return -fuse.EIO, 0
	}

	// stat to obtain size
	h := uint64(0)
	if fi, err := f.client.Stat(path); err == nil {
		h = f.NewHandle(path, casters.FileInfoCast(fi))
	}

	// allocate handle using caller flags (so writes are allowed if requested)
	fmt.Printf("[log] (Create): returning handle=%d path=%s flags=%#o mode=%#o\n", h, path, flags, mode)

	// No preloading required: writes are sent directly on Write.

	return 0, h
}
