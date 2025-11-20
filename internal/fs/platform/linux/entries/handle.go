//go:build linux

package entries

import (
	"context"
	"hash/crc32"
	"os"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/flags"
	"github.com/mimic/internal/core/helpers"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/interfaces"
)

type Handle struct {
	path   string
	wc     interfaces.WebClient
	logger logger.FullLogger
	flags  flags.OpenFlag

	mu       sync.Mutex
	segments map[int64][]byte
	dirty    bool
}

func (h *Handle) Attr(ctx context.Context, a *fuse.Attr) error {
	fi, err := h.wc.Stat(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		return err
	}

	attr := casters.FileInfoCast(fi)
	attr.Inode = uint64(crc32.ChecksumIEEE([]byte(h.path)) + 1)

	*a = *attr
	return nil
}

func (h *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	if !h.flags.ReadAllowed() {
		return nil, syscall.Errno(syscall.EACCES)
	}

	data, err := h.wc.Read(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
		return nil, err
	}
	return data, nil
}

func (h *Handle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if !h.flags.WriteAllowed() {
		return syscall.Errno(syscall.EACCES)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.segments == nil {
		h.segments = make(map[int64][]byte)
	}

	data := make([]byte, len(req.Data))
	copy(data, req.Data)
	h.segments[req.Offset] = data
	h.dirty = true

	resp.Size = len(req.Data)
	return nil
}

func (h *Handle) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	if !h.flags.WriteAllowed() {
		return syscall.Errno(syscall.EACCES)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.dirty || len(h.segments) == 0 {
		return nil
	}

	base, err := h.wc.Read(h.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		base = []byte{}
	}

	merged := helpers.MergeSegmentsInto(base, h.segments)

	h.logger.Logf("Fsync called to flush: %s", h.path)

	if err := h.wc.Write(h.path, merged); err != nil {
		return err
	}

	h.segments = nil
	h.dirty = false
	return nil
}

func (h *Handle) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return h.Fsync(ctx, &fuse.FsyncRequest{
		Handle: req.Handle,
	})
}

func (h *Handle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return h.Fsync(ctx, &fuse.FsyncRequest{
		Handle: req.Handle,
	})
}

func (h *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	h.logger.Logf("ReadDirAll called for %s", h.path)

	ents, err := h.wc.ReadDir(h.path)
	if err != nil {
		return nil, syscall.Errno(syscall.ENOENT)
	}

	var dirents []fuse.Dirent
	for _, fi := range ents {
		var dtype fuse.DirentType
		if fi.IsDir() {
			dtype = fuse.DT_Dir
		} else {
			dtype = fuse.DT_File
		}

		childPath := h.path + fi.Name()

		dirents = append(dirents, fuse.Dirent{
			Inode: uint64(crc32.ChecksumIEEE([]byte(childPath)) + 1),
			Name:  fi.Name(),
			Type:  dtype,
		})
	}

	return dirents, nil
}
