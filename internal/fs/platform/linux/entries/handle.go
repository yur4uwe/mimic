//go:build linux

package entries

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/interfaces"
)

type Handle struct {
	path string
	wc   interfaces.WebClient

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

func (h *Handle) mergeSegmentsInto(base []byte) []byte {
	maxEnd := int64(len(base))
	for off, seg := range h.segments {
		end := off + int64(len(seg))
		if end > maxEnd {
			maxEnd = end
		}
	}

	merged := make([]byte, maxEnd)
	copy(merged, base)

	for off, seg := range h.segments {
		copy(merged[off:], seg)
	}

	return merged
}

func (h *Handle) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
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

	merged := h.mergeSegmentsInto(base)

	fmt.Println("Fsync called to flush:", h.path)

	if err := h.wc.Write(h.path, merged); err != nil {
		return err
	}

	h.segments = nil
	h.dirty = false
	return nil
}

func (h *Handle) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return h.Fsync(ctx, &fuse.FsyncRequest{})
}

func (h *Handle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if err := h.Fsync(ctx, &fuse.FsyncRequest{}); err != nil {
		return err
	}
	return nil
}

func (h *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll called for", h.path)

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
