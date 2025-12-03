//go:build linux

package entries

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"syscall"

	"bazil.org/fuse"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/flags"
	"github.com/mimic/internal/core/locking"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/fs/common"
	"github.com/mimic/internal/interfaces"
)

type Handle struct {
	common.FileHandle
	wc     interfaces.WebClient
	logger logger.FullLogger
}

func NewHandle(wc interfaces.WebClient, logger logger.FullLogger, path string, flags flags.OpenFlag) *Handle {
	return &Handle{
		wc:         wc,
		logger:     logger,
		FileHandle: *common.NewFilehandle(path, flags),
	}
}

func (h *Handle) Attr(ctx context.Context, a *fuse.Attr) error {
	h.logger.Logf("(Handle) [Attr] called for %s", h.Path())
	fi, err := h.wc.Stat(h.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [Attr] Stat error for %s: %v; returning EIO", h.Path(), err)
		return syscall.Errno(syscall.EIO)
	}

	attr := casters.FileInfoCast(fi)
	attr.Inode = uint64(crc32.ChecksumIEEE([]byte(h.Path())) + 1)

	*a = *attr
	return nil
}

func (h *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	h.logger.Logf("(Handle) [ReadAll] called for %s", h.Path())
	if !h.Flags().ReadAllowed() {
		h.logger.Logf("(Handle) [ReadAll] access denied for %s, flag state: %+v", h.Path(), h.Flags())
		return nil, syscall.Errno(syscall.EACCES)
	}

	data, err := h.wc.Read(h.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [ReadAll] Read error for %s: %v; returning EIO", h.Path(), err)
		return nil, syscall.Errno(syscall.EIO)
	}
	return data, nil
}

func (h *Handle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	h.logger.Logf("(Handle) [Read] called for %s, offset: %d, size: %d", h.Path(), req.Offset, req.Size)
	if !h.Flags().ReadAllowed() {
		h.logger.Logf("(Handle) [Read] access denied for %s, flag state: %+v", h.Path(), h.Flags())
		return syscall.Errno(syscall.EACCES)
	}

	data, err := h.wc.Read(h.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		h.logger.Logf("(Handle) [Read] Read error for %s: %v; returning EIO", h.Path(), err)
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
	h.logger.Logf("(Handle) [Write] called for %s, offset: %d, size: %d", h.Path(), req.Offset, len(req.Data))
	if !h.Flags().WriteAllowed() {
		h.logger.Logf("(Handle) [Write] access denied for %s, flag state: %+v", h.Path(), h.Flags())
		return syscall.Errno(syscall.EACCES)
	}

	h.MLock()
	h.AddToBuffer(req.Offset, req.Data)
	h.MUnlock()

	resp.Size = len(req.Data)
	return nil
}

func (h *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	h.logger.Logf("(Handle) [ReadDirAll] called for %s", h.Path())

	ents, err := h.wc.ReadDir(h.Path())
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

		childPath := h.Path() + fi.Name()

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
	h.logger.Logf("(Handle) [Flush] called for %s", h.Path())
	if !h.Flags().WriteAllowed() {
		h.logger.Logf("(Handle) [Flush] readonly %s, flag state: %+v", h.Path(), h.Flags())
		return nil
	}

	h.MLock()
	buf, off := h.Buffer()
	if buf != nil {
		// Attempt to write the anchored buffer using WriteOffset. The
		// underlying wrapper may try a partial PUT and fall back to a
		// merged full-write if necessary. If the remote reports the file
		// doesn't exist and this handle was opened with create, build a
		// full-sized buffer (zeros up to offset) and write it.
		if err := h.wc.WriteOffset(h.Path(), buf, off); err != nil {
			if os.IsNotExist(err) && h.Flags().Create() {
				end := off + int64(len(buf))
				if end > int64(^uint(0)>>1) {
					h.logger.Logf("(Handle) [Flush] too large allocate for %s; returning EIO", h.Path())
					h.MUnlock()
					return syscall.Errno(syscall.EIO)
				}
				full := make([]byte, int(end))
				copy(full[int(off):], buf)
				if werr := h.wc.Write(h.Path(), full); werr != nil {
					h.logger.Logf("(Handle) [Flush] client.Write error for %s: %v; returning EIO", h.Path(), werr)
					h.MUnlock()
					return syscall.Errno(syscall.EIO)
				}
			} else {
				h.logger.Logf("(Handle) [Flush] client.WriteOffset error for %s: %v; returning EIO", h.Path(), err)
				h.MUnlock()
				return syscall.Errno(syscall.EIO)
			}
		}
		// clear buffer on success
		h.ClearBuffer()
	}
	h.MUnlock()
	return nil
}

func (h *Handle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	h.logger.Logf("(Handle) [Release] called for %s", h.Path())

	if h.IsDirty() {
		h.logger.Logf("(Handle) [Release] flushing buffer for %s", h.Path())
		if err := h.Flush(ctx, nil); err != nil {
			h.logger.Logf("(Handle) [Release] flush error for %s: %v", h.Path(), err)
			return err
		}
	}

	return nil
}

func getLockOwner(reqOwner fuse.LockOwner) []byte {
	if reqOwner != 0 {
		return []byte(fmt.Sprintf("%d", reqOwner))
	}
	return []byte(fmt.Sprintf("pid:%d", os.Getpid()))
}

func (h *Handle) Lock(ctx context.Context, req *fuse.LockRequest) error {
	owner := getLockOwner(req.LockOwner)

	start := req.Lock.Start
	end := req.Lock.End
	ltype := locking.LockType(req.Lock.Type)

	if err := h.wc.Lock(h.Path(), owner, start, end, ltype); err != nil {
		if err == locking.ErrWouldBlock {
			return syscall.Errno(syscall.EAGAIN)
		}
		return syscall.Errno(syscall.EIO)
	}
	return nil
}

func (h *Handle) LockWait(ctx context.Context, req *fuse.LockWaitRequest) error {
	owner := getLockOwner(req.LockOwner)

	start := req.Lock.Start
	end := req.Lock.End
	// derive locking type from incoming request; default to write lock
	ltype := locking.LockType(req.Lock.Type)
	if ltype == 0 {
		ltype = locking.F_WRLCK
	}

	if err := h.wc.LockWait(ctx, h.Path(), owner, start, end, ltype); err != nil {
		return syscall.Errno(syscall.EIO)
	}
	return nil
}

func (h *Handle) Unlock(ctx context.Context, req *fuse.UnlockRequest) error {
	owner := getLockOwner(req.LockOwner)

	start := req.Lock.Start
	end := req.Lock.End

	if err := h.wc.Unlock(h.Path(), owner, start, end); err != nil {
		if err == locking.ErrNotOwner {
			return syscall.Errno(syscall.EACCES)
		}
		return syscall.Errno(syscall.EIO)
	}
	return nil
}

// QueryLock returns the current state of locks held for the byte
// range of the node.
//
// See QueryLockRequest for details on how to respond.
//
// To simplify implementing this method, resp.Lock is prefilled to
// have Lock.Type F_UNLCK, and the whole struct should be
// overwritten for in case of conflicting locks.
func (h *Handle) QueryLock(ctx context.Context, req *fuse.QueryLockRequest, resp *fuse.QueryLockResponse) error {
	start := req.Lock.Start
	end := req.Lock.End

	lock := h.wc.Query(h.Path(), start, end)
	if lock == nil {
		return nil
	}

	resp.Lock.Type = fuse.LockType(lock.Type)
	resp.Lock.PID = int32(lock.PID)
	resp.Lock.Start = start
	resp.Lock.End = end
	return nil
}
