package common

import (
	"sync"

	"github.com/mimic/internal/core/flags"
)

type FileHandle struct {
	path  string
	flags flags.OpenFlag

	mu     sync.Mutex
	buffer []byte
	offset int64
}

func NewFilehandle(path string, oflags flags.OpenFlag) *FileHandle {
	return &FileHandle{
		path:  path,
		flags: oflags,
	}
}

func (fh *FileHandle) Lock() {
	fh.mu.Lock()
}

func (fh *FileHandle) Unlock() {
	fh.mu.Unlock()
}

func (fh *FileHandle) AddToBuffer(offset int64, data []byte) {
	if len(data) == 0 {
		return
	}
	if fh.buffer == nil {
		fh.offset = offset
		fh.buffer = make([]byte, len(data))
		copy(fh.buffer, data)
		return
	}

	if offset < fh.offset {
		shift := fh.offset - offset
		newLen := int(shift) + len(fh.buffer)
		newBuf := make([]byte, newLen)
		copy(newBuf[int(shift):], fh.buffer)
		fh.buffer = newBuf
		fh.offset = offset
	}

	rel := int(offset - fh.offset)
	end := rel + len(data)
	if end > len(fh.buffer) {
		nb := make([]byte, end)
		copy(nb, fh.buffer)
		fh.buffer = nb
	}
	copy(fh.buffer[rel:end], data)
}

func (fh *FileHandle) ClearBuffer() {
	fh.buffer = nil
}

func (fh *FileHandle) Buffer() ([]byte, int64) {
	return fh.buffer, fh.offset
}

func (fh *FileHandle) IsDirty() bool {
	return fh.buffer != nil
}

func (fh *FileHandle) Flags() flags.OpenFlag {
	return fh.flags
}

func (fh *FileHandle) Path() string {
	return fh.path
}
