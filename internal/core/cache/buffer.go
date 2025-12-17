package cache

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNegativeLength = errors.New("negative length")
	ErrNegativeOffset = errors.New("negative offset")
	ErrOutOfBounds    = errors.New("read/write out of bounds")
)

type BufferSnapshot struct {
	Data []byte
	Base int64
	Mask Mask
}

// FileBuffer represents a file image kept in memory for a mapped path.
type FileBuffer struct {
	BufferSnapshot
	mu          sync.RWMutex
	Dirty       bool
	HandleCount int
}

func (fb *FileBuffer) BasePos() int64 {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.Base
}

func (fb *FileBuffer) CopyBuffer() *BufferSnapshot {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	if len(fb.Data) == 0 {
		return &BufferSnapshot{Data: nil, Base: fb.Base, Mask: nil}
	}
	cp := make([]byte, len(fb.Data))
	copy(cp, fb.Data)
	return &BufferSnapshot{Data: cp, Base: fb.Base, Mask: fb.Mask}
}

func (fb *FileBuffer) SetBase(b int64) {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.Base = b
}

func (fb *FileBuffer) ReadAt(offset int64, length int) ([]byte, error) {
	if length < 0 {
		return nil, ErrNegativeLength
	}
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	if offset < 0 {
		return nil, ErrNegativeOffset
	}
	end := offset + int64(length)
	if end > int64(len(fb.Data)) {
		return nil, ErrOutOfBounds
	}

	out := make([]byte, length)
	copy(out, fb.Data[offset:end])
	return out, nil
}

func (fb *FileBuffer) String() string {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fmt.Sprintf("Buffer: dirty=%v base=%d len=%d", fb.Dirty, fb.Base, len(fb.Data))
}

// WriteAt writes data at the given offset, growing the buffer if needed.
// It sets the Dirty flag.
func (fb *FileBuffer) WriteAt(offset int64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if offset < 0 {
		return ErrNegativeOffset
	}
	fb.mu.Lock()
	defer fb.mu.Unlock()

	// no data yet, create new buffer
	if len(fb.Data) == 0 {
		fb.Base = offset
		fb.Data = make([]byte, len(data))
		copy(fb.Data, data)
		fb.Mask = make(Mask, maskSize(int64(len(data))))
		fb.Mask.smearPages(0, int64(len(data)))
		fb.Dirty = true
		return nil
	}

	// if offset is within current data range
	relStart := offset - fb.Base

	// within current data
	if relStart >= 0 {
		// calculate end within buffer
		end := relStart + int64(len(data))
		if end > int64(len(fb.Data)) {
			// grow if new end exceeds current size
			newData := make([]byte, end)
			copy(newData, fb.Data)
			fb.Data = newData
		}
		fb.Mask.smearPages(relStart, end)
		copy(fb.Data[relStart:end], data)
		fb.Dirty = true
		return nil
	}

	// relStart < 0: incoming write starts before current base; prepend.
	// Calculate how many bytes we need to prepend.
	prepend := int64(0 - relStart)
	newLen := prepend + int64(len(fb.Data))
	newData := make([]byte, newLen)
	// copy incoming data at offset 0
	copy(newData[0:len(data)], data)
	// copy existing data after the prepend region
	copy(newData[prepend:newLen], fb.Data)

	fb.Mask = fb.Mask.shiftedRight(prepend, int64(len(newData)))
	fb.Mask.smearPages(0, int64(len(data)))

	fb.Base = offset
	fb.Data = newData
	fb.Dirty = true
	return nil
}

func (fb *FileBuffer) Clear() {
	fb.mu.Lock()
	fb.Data = nil
	fb.Dirty = false
	fb.Mask.clear()
	fb.mu.Unlock()
}

func (fb *FileBuffer) IsValidAt(i int64) bool {
	if i < fb.Base || i >= fb.Base+int64(len(fb.Data)) {
		return false
	}
	return fb.Mask.IsDirty(i - fb.Base)
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

func (fb *FileBuffer) DirtyRange(start, length int64) bool {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	for i := start; i < start+length; i++ {
		if !fb.Mask.IsDirty(i) {
			return false
		}
	}
	return true
}
