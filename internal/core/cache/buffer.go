package cache

import (
	"errors"
	"sync"
)

var (
	NegativeLengthError = errors.New("negative length")
	NegativeOffsetError = errors.New("negative offset")
	OutOfBoundsError    = errors.New("read/write out of bounds")
)

// FileBuffer represents a file image kept in memory for a mapped path.
type FileBuffer struct {
	mu          sync.RWMutex
	Base        int64
	Data        []byte
	Dirty       bool
	HandleCount int
}

func (fb *FileBuffer) BasePos() int64 {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.Base
}

func (fb *FileBuffer) CopyBuffer() ([]byte, int64) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	if len(fb.Data) == 0 {
		return nil, fb.Base
	}
	cp := make([]byte, len(fb.Data))
	copy(cp, fb.Data)
	return cp, fb.Base
}

func (fb *FileBuffer) SetBase(b int64) {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.Base = b
}

func (fb *FileBuffer) ReadAt(offset int64, length int) ([]byte, error) {
	if length < 0 {
		return nil, NegativeLengthError
	}
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	if offset < 0 {
		return nil, NegativeOffsetError
	}
	end := offset + int64(length)
	if end > int64(len(fb.Data)) {
		return nil, OutOfBoundsError
	}

	out := make([]byte, length)
	copy(out, fb.Data[offset:end])
	return out, nil
}

// WriteAt writes data at the given offset, growing the buffer if needed.
// It sets the Dirty flag.
func (fb *FileBuffer) WriteAt(offset int64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if offset < 0 {
		return NegativeOffsetError
	}
	fb.mu.Lock()
	defer fb.mu.Unlock()

	end := offset + int64(len(data))
	if end > int64(len(fb.Data)) {
		newData := make([]byte, end)
		copy(newData, fb.Data)
		fb.Data = newData
	}
	copy(fb.Data[offset:end], data)
	fb.Dirty = true
	return nil
}

func (fb *FileBuffer) Clear() {
	fb.mu.Lock()
	fb.Data = nil
	fb.Dirty = false
	fb.mu.Unlock()
}

func (fb *FileBuffer) Size() int64 {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return int64(len(fb.Data))
}

func (fb *FileBuffer) MarkClean() {
	fb.mu.Lock()
	fb.Dirty = false
	fb.mu.Unlock()
}

func (fb *FileBuffer) IncHandle() {
	fb.mu.Lock()
	fb.HandleCount++
	fb.mu.Unlock()
}

func (fb *FileBuffer) DecHandle() {
	fb.mu.Lock()
	if fb.HandleCount > 0 {
		fb.HandleCount--
	}
	fb.mu.Unlock()
}

// BufferCache stores FileBuffer entries by path.
type BufferCache struct {
	entries sync.Map // map[string]*FileBuffer
}

func NewBufferCache() *BufferCache {
	return &BufferCache{}
}

// Get returns the buffer for a path if present.
func (bc *BufferCache) Get(path string) (*FileBuffer, bool) {
	value, ok := bc.entries.Load(path)
	if !ok {
		return nil, false
	}
	buf, ok := value.(*FileBuffer)
	return buf, ok
}

// GetOrCreate returns the buffer for a path, creating it if missing.
func (bc *BufferCache) GetOrCreate(path string) *FileBuffer {
	if v, ok := bc.entries.Load(path); ok {
		if fb, ok := v.(*FileBuffer); ok {
			return fb
		}
	}

	fb := &FileBuffer{
		Data:        make([]byte, 0),
		HandleCount: 1,
	}
	actual, _ := bc.entries.LoadOrStore(path, fb)
	return actual.(*FileBuffer)
}

// Set stores the provided buffer for the path (overwrites).
func (bc *BufferCache) Set(path string, buffer *FileBuffer) {
	bc.entries.Store(path, buffer)
}

func (bc *BufferCache) Delete(path string) {
	bc.entries.Delete(path)
}
