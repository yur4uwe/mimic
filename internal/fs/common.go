package fs

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/flags"
	fuselib "github.com/winfsp/cgofuse/fuse"
)

const (
	READAHEAD_DEFAULT int64 = 64 * 1024
	READAHEAD_MAX     int64 = 4 * 1024 * 1024
)

type FileHandle struct {
	path  string
	flags flags.OpenFlag
	stat  *fuselib.Stat_t

	remoteSize   int64
	lastReadSize int64

	mu     sync.Mutex
	buffer *cache.FileBuffer
}

func NewFilehandle(path string, oflags flags.OpenFlag, stat *fuselib.Stat_t) *FileHandle {
	var remoteSize int64 = 0
	if stat != nil {
		remoteSize = stat.Size
	}
	return &FileHandle{
		path:       path,
		flags:      oflags,
		stat:       stat,
		remoteSize: remoteSize,
	}
}

func (fh *FileHandle) MLock() {
	fh.mu.Lock()
}

func (fh *FileHandle) MUnlock() {
	fh.mu.Unlock()
}

func (fh *FileHandle) Size() int64 {
	if fh.buffer == nil {
		fmt.Printf("[File Size] buffer is null returning stat.Size = %d\n", fh.stat.Size)
		return fh.stat.Size
	}

	fh.buffer.MRLock()
	defer fh.buffer.MRUnlock()
	// I don't know if this would work if file is downsized
	fmt.Printf("[File Size] buffer %p is present for %s returning max(buffer.Length()+buffer.BasePos(), stat.Size) = max(%d+%d, %d)\n", fh.buffer, fh.path, fh.buffer.Length(), fh.buffer.BasePos(), fh.stat.Size)
	return max(fh.buffer.Length()+fh.buffer.BasePos(), fh.stat.Size)
}

func (fh *FileHandle) AddToBuffer(offset int64, data []byte) {
	if len(data) == 0 {
		return
	}
	if fh.buffer == nil {
		// defensive: create per-handle buffer if not set (should be set by fs.NewHandle)
		fh.buffer = &cache.FileBuffer{}
		fh.buffer.Data = make([]byte, 0)
		fh.buffer.IncHandle()
	}
	// Use absolute offsets (current code treats buffer Data[0] as file offset 0).
	_ = fh.buffer.WriteAt(offset, data)
}

// AddRemoteToBuffer inserts data fetched from remote into the per-handle buffer
// without marking the buffer as dirty and without overwriting pages that are
// already present.
func (fh *FileHandle) AddRemoteToBuffer(offset int64, data []byte) {
	if len(data) == 0 {
		return
	}
	if fh.buffer == nil {
		fh.buffer = &cache.FileBuffer{}
		fh.buffer.Data = make([]byte, 0)
		fh.buffer.IncHandle()
	}
	_ = fh.buffer.WriteRemoteAt(offset, data)
}

// ClearBuffer clears the shared buffer (used after a successful Flush/Release).
func (fh *FileHandle) ClearBuffer() {
	if fh.buffer == nil {
		return
	}
	fh.buffer.Clear()
	fh.buffer.DecHandle()
	fh.buffer = nil
}

// nil, 0 if no buffer
func (fh *FileHandle) CopyBuffer() *cache.BufferSnapshot {
	if fh.buffer == nil {
		return &cache.BufferSnapshot{}
	}
	return fh.buffer.CopyBuffer()
}

func (fh *FileHandle) IsDirty() bool {
	return fh.buffer != nil && fh.buffer.Dirty
}

func (fh *FileHandle) PageAlignedRange(offset, length int64) (int64, int64) {
	reqPageStart := offset - (offset % cache.PageSize)
	reqPagesCount := (length + (offset - reqPageStart) + cache.PageSize - 1) / cache.PageSize
	readAheadLen := reqPagesCount * cache.PageSize

	readAheadLen = max(readAheadLen, READAHEAD_DEFAULT)
	if fh.lastReadSize != 0 && readAheadLen < fh.lastReadSize*2 {
		readAheadLen = fh.lastReadSize * 2
	}
	readAheadLen = min(readAheadLen, READAHEAD_MAX)

	fh.lastReadSize = readAheadLen

	if reqPageStart+readAheadLen > fh.remoteSize {
		readAheadLen = max(fh.remoteSize-reqPageStart, 0)
	}

	return reqPageStart, readAheadLen
}

func (fs *FuseFS) NewHandle(path string, stat *fuselib.Stat_t, oflags uint32) uint64 {
	file_handle := atomic.AddUint64(&fs.nextHandle, 1)
	fh := NewFilehandle(path, flags.OpenFlag(oflags), stat)

	fb := fs.bufferCache.GetOrCreate(path)
	fs.logger.Logf("[NewHandle] created handle=%d for path=%s with buffer %p", file_handle, path, fb)
	fb.IncHandle()
	fh.buffer = fb

	fs.handles.Store(file_handle, fh)
	return file_handle
}

func (fs *FuseFS) GetHandle(handle uint64) (*FileHandle, bool) {
	file, ok := fs.handles.Load(handle)
	if !ok {
		return nil, false
	}
	of := file.(*FileHandle)
	return of, true
}

func (fs *FuseFS) ReleaseHandle(handle uint64) {
	fh, ok := fs.GetHandle(handle)
	if !ok {
		return
	}
	if fh.buffer != nil {
		fh.buffer.DecHandle()
		fh.buffer = nil
	}
	fs.handles.Delete(handle)
}
