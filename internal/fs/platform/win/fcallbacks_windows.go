package win

import (
	"fmt"
	"os"
	"strings"

	"github.com/winfsp/cgofuse/fuse"
)

const (
	EACCES = fuse.EACCES
)

func (f *WinfspFS) Access(p string, mask uint32) int {
	fmt.Println("Access called for", p, "mask:", mask)
	// simple existence check for now
	if strings.HasSuffix(p, "/") && p != "/" {
		// try without trailing slash
		p = strings.TrimSuffix(p, "/")
	}
	_, err := f.client.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return -ENOENT
		}
		return -EIO
	}
	// success -> return 0 (no errno)
	return 0
}

func (f *WinfspFS) Flush(p string, fh uint64) int {
	fmt.Println("Flush called for", p, "fh:", fh)
	// best-effort: call Fsync which may be implemented to commit per-handle data
	return f.Fsync(p, false, fh)
}

func (f *WinfspFS) Fsync(p string, datasync bool, fh uint64) int {
	// If you add per-handle write buffers, commit them here.
	// For now do nothing (no-op) and return success.
	fmt.Println("Fsync called for", p, "fh:", fh, "datasync:", datasync)

	file, ok := f.GetHandle(fh)
	if !ok {
		return -EIO
	}

	file.mu.Lock()
	defer file.mu.Unlock()

	if !file.dirty {
		return 0 // nothing to do
	}

	if err := f.client.WriteStreamRange(file.path, strings.NewReader(string(file.buf)), 0); err != nil {
		fmt.Println("Fsync: WriteStreamRange error:", err)
		return -EIO
	}

	file.dirty = false
	return 0
}

func (f *WinfspFS) Mknod(path string, mode uint32, dev uint64) int {
	fmt.Println("Mknod called for", path, "mode:", mode, "dev:", dev)

	// Only support regular file creation via mknod; other node types are unsupported.
	if mode&fuse.S_IFMT != fuse.S_IFREG {
		fmt.Println("Mknod: only regular files supported")
		return -ENOSYS
	}

	// normalize path
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// create an empty file on remote (PUT)
	if err := f.client.Create(path); err != nil {
		fmt.Println("Mknod: remote write failed:", err)
		return -EIO
	}

	// success
	return 0
}

func (f *WinfspFS) Readlink(path string) (int, string) {
	fmt.Println("Readlink called for", path)
	// return symlink target
	return -ENOSYS, ""
}

func (f *WinfspFS) Truncate(p string, size int64, fh uint64) int {
	fmt.Println("Truncate called for:", p, "fh:", fh, "size:", size)
	// implement truncate by reading whole file, resizing, and PUTting back.
	// This is inefficient but works with servers that only support full PUT.
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}

	data, err := f.client.Read(p)
	if err != nil {
		// if file doesn't exist, report ENOENT
		if os.IsNotExist(err) {
			return -ENOENT
		}
		return -EIO
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
		return -EIO
	}

	return 0
}

func (f *WinfspFS) Unlink(p string) int {
	fmt.Println("Unlink called for", p)
	// delete file
	if strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	if err := f.client.Remove(p); err != nil {
		return -EIO
	}

	return 0
}

func (f *WinfspFS) Write(path string, buffer []byte, offset int64, file_handle uint64) int {
	fmt.Println("Write called for", path, "fh:", file_handle, "offset:", offset, "len:", len(buffer))

	file, ok := f.GetHandle(file_handle)
	if !ok {
		return -EIO
	}

	file.mu.Lock()
	defer file.mu.Unlock()

	needed := int(offset) + len(buffer)
	if len(file.buf) < needed {
		newbuf := make([]byte, needed)
		copy(newbuf, file.buf)
		file.buf = newbuf
	}
	copy(file.buf[offset:], buffer)
	file.dirty = true
	if int64(len(file.buf)) > file.size {
		file.size = int64(len(file.buf))
	}

	return len(buffer)
}

func (f *WinfspFS) Create(path string, flags int, mode uint32) (int, uint64) {
	fmt.Println("Create called for", path, "flags:", flags, "mode:", mode)

	// normalize
	if strings.HasSuffix(path, "/") && path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	// create empty remote file (PUT). Use wrapper which updates cache.
	if err := f.client.Write(path, []byte{}); err != nil {
		fmt.Println("Create: remote write failed:", err)
		// map common remote permission errors where possible
		if os.IsPermission(err) {
			return -EACCES, 0
		}
		return -EIO, 0
	}

	// stat to obtain size
	var sz int64
	if fi, err := f.client.Stat(path); err == nil {
		sz = fi.Size()
	}

	// allocate handle using caller flags (so writes are allowed if requested)
	h := f.NewHandle(path, flags, sz)
	fmt.Println("Create returning handle:", h)

	// If opened for write, preload buffer so Write can patch it
	if flags&(fuse.O_WRONLY|fuse.O_RDWR) != 0 {
		if of, ok := f.GetHandle(h); ok {
			if data, err := f.client.Read(path); err == nil {
				of.mu.Lock()
				of.buf = data
				of.size = int64(len(data))
				of.mu.Unlock()
				fmt.Printf("Create: preloaded %d bytes into handle=%d\n", len(data), h)
			} else {
				fmt.Println("Create: preload read failed:", err)
			}
		}
	}

	return 0, h
}
