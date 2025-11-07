//go:build linux || darwin

package entries

import (
	"context"
	"hash/crc32"
	"os"
	"syscall"

	"bazil.org/fuse"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/interfaces"
)

type File struct {
	path string
	wc   interfaces.WebClient
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fi, err := f.wc.Stat(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		return err
	}

	attr := casters.FileInfoCast(fi)
	attr.Inode = uint64(crc32.ChecksumIEEE([]byte(f.path)) + 1)

	*a = *attr
	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	data, err := f.wc.Read(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
		return nil, err
	}
	return data, nil
}
