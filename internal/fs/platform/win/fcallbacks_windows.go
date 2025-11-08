package win

import (
	"fmt"
	"os"
	"strings"
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
	return 0
}

func (f *WinfspFS) Chmod(path string, mode uint32) int {
	fmt.Println("Chmod called for", path, "mode:", mode)
	return -EIO
}

func (f *WinfspFS) Chown(path string, uid uint32, gid uint32) int {
	fmt.Println("Chown called for", path, "uid:", uid, "gid:", gid)
	return -EIO
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
	return 0
}

func (f *WinfspFS) Mknod(path string, mode uint32, dev uint64) int {
	fmt.Println("Mknod called for", path, "mode:", mode, "dev:", dev)
	// create special file (usually not needed for WebDAV)
	return -ENOSYS
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

// xattr helpers for files (implement only if needed)
func (f *WinfspFS) Getxattr(path string, name string) (int, []byte) {
	fmt.Println("Getxattr called for", path, "name:", name)
	return -EIO, nil
}

func (f *WinfspFS) Setxattr(path string, name string, value []byte, flags int) int {
	fmt.Println("Setxattr called for", path, "name:", name, "flags:", flags)
	return -EIO
}

func (f *WinfspFS) Listxattr(path string, fill func(name string) bool) int {
	fmt.Println("Listxattr called for", path)
	return -EIO
}

func (f *WinfspFS) Removexattr(path string, name string) int {
	fmt.Println("Removexattr called for", path, "name:", name)
	return -EIO
}
