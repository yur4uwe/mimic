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
	buffer []byte

	mu     sync.Mutex
	client interfaces.WebClient
}

func NewHandle(wc interfaces.WebClient, logger logger.FullLogger, path string, flags flags.OpenFlag, client interfaces.WebClient) *Handle {
	return &Handle{
		wc:     wc,
		logger: logger,
		flags:  flags,
		path:   path,
		client: client,
	}
}

func (h *Handle) Attr(ctx context.Context, a *fuse.Attr) error {
	h.logger.Logf("(Handle) [Attr] called for %s", h.path)
	fi, err := h.wc.Stat(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [Attr] Stat error for %s: %v; returning EIO", h.path, err)
		return syscall.Errno(syscall.EIO)
	}

	attr := casters.FileInfoCast(fi)
	attr.Inode = uint64(crc32.ChecksumIEEE([]byte(h.path)) + 1)

	*a = *attr
	return nil
}

func (h *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	h.logger.Logf("(Handle) [ReadAll] called for %s", h.path)
	if !h.flags.ReadAllowed() {
		h.logger.Logf("(Handle) [ReadAll] access denied for %s, flag state: %+v", h.path, h.flags)
		return nil, syscall.Errno(syscall.EACCES)
	}

	data, err := h.wc.Read(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [ReadAll] Read error for %s: %v; returning EIO", h.path, err)
		return nil, syscall.Errno(syscall.EIO)
	}
	return data, nil
}

func (h *Handle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	h.logger.Logf("(Handle) [Read] called for %s, offset: %d, size: %d", h.path, req.Offset, req.Size)
	if !h.flags.ReadAllowed() {
		h.logger.Logf("(Handle) [Read] access denied for %s, flag state: %+v", h.path, h.flags)
		return syscall.Errno(syscall.EACCES)
	}

	data, err := h.wc.Read(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [Read] Read error for %s: %v; returning EIO", h.path, err)
		return syscall.Errno(syscall.EIO)
	}

	if req.Offset >= int64(len(data)) {
		resp.Data = []byte{}
		return nil
	}

	if req.Offset+int64(req.Size) > int64(len(data)) {
		req.Size = int(len(data) - int(req.Offset))
	}

	resp.Data = make([]byte, req.Size)

	copy(resp.Data, data[req.Offset:req.Offset+int64(req.Size)])

	return nil
}

func (h *Handle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	h.logger.Logf("(Handle) [Write] called for %s, offset: %d, size: %d", h.path, req.Offset, len(req.Data))
	if !h.flags.WriteAllowed() {
		h.logger.Logf("(Handle) [Write] access denied for %s, flag state: %+v", h.path, h.flags)
		return syscall.Errno(syscall.EACCES)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.buffer = helpers.AddToBuffer(h.buffer, req.Offset, req.Data)

	resp.Size = len(req.Data)
	return nil
}

func (h *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	h.logger.Logf("(Handle) [ReadDirAll] called for %s", h.path)

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

func (h *Handle) Poll(ctx context.Context, req *fuse.PollRequest, resp *fuse.PollResponse) error {
	return syscall.Errno(syscall.ENOSYS)
}

func (h *Handle) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	h.logger.Logf("(Handle) [Flush] called for %s", h.path)
	if !h.flags.WriteAllowed() {
		h.logger.Logf("(Handle) [Flush] readonly %s, flag state: %+v", h.path, h.flags)
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.buffer != nil {
		base, err := h.client.Read(h.path)
		if err != nil {
			// if not exist, treat base as empty only if opened with create flag
			if os.IsNotExist(err) && h.flags.Create() {
				base = []byte{}
			} else {
				h.logger.Logf("(Handle) [Flush] error reading base for %s: %v; returning EIO", h.path, err)
				return syscall.Errno(syscall.EIO)
			}
		}

		merged := helpers.MergeBufferIntoBase(base, h.buffer)

		err = h.client.Write(h.path, merged)
		if err != nil {
			h.logger.Logf("(Handle) [Flush] client.Write error for %s: %v; returning EIO", h.path, err)
			return syscall.Errno(syscall.EIO)
		}
	}

	h.buffer = nil
	return nil
}

func (h *Handle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	h.logger.Logf("(Handle) [Release] called for %s", h.path)

	if h.buffer != nil {
		h.logger.Logf("(Handle) [Release] flushing buffer for %s", h.path)
		if err := h.Flush(ctx, nil); err != nil {
			h.logger.Logf("(Handle) [Release] flush error for %s: %v", h.path, err)
			return err
		}
	}

	return nil
}
