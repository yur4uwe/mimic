package cache

import "sync"

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
