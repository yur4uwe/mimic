package fs

import (
	"sync"
	"sync/atomic"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/flags"
	fuselib "github.com/winfsp/cgofuse/fuse"
)

type FileHandle struct {
	path       string
	flags      flags.OpenFlag
	stat       *fuselib.Stat_t
	remoteSize int64

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

func (fh *FileHandle) AddToBuffer(offset int64, data []byte) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
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
	fh.mu.Lock()
	defer fh.mu.Unlock()
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

func (fh *FileHandle) Flags() flags.OpenFlag {
	return fh.flags
}

func (fh *FileHandle) Path() string {
	return fh.path
}

func (fs *FuseFS) NewHandle(path string, stat *fuselib.Stat_t, oflags uint32) uint64 {
	file_handle := atomic.AddUint64(&fs.nextHandle, 1)
	fh := NewFilehandle(path, flags.OpenFlag(oflags), stat)

	fb := fs.bufferCache.GetOrCreate(path)
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
