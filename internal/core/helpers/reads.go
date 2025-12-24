package helpers

import (
	"github.com/mimic/internal/core/cache"
)

const (
	READAHEAD_DEFAULT int64 = 8 * 1024  // 8 KB
	READAHEAD_MAX     int64 = 64 * 1024 // 64 KB cap
)

func PageAlignedRange(offset, length, remoteSize int64) (int64, int64) {
	reqPageStart := offset - (offset % cache.PageSize)
	reqPagesCount := (length + (offset - reqPageStart) + cache.PageSize - 1) / cache.PageSize
	readAheadLen := reqPagesCount * cache.PageSize

	// compute readahead length: at least requested length, at least READAHEAD_DEFAULT,
	// but capped to READAHEAD_MAX and remote size when known
	readAheadLen = min(max(readAheadLen, int64(READAHEAD_DEFAULT)), READAHEAD_MAX)

	// don't read past known remote end
	if reqPageStart+readAheadLen > remoteSize {
		readAheadLen = max(remoteSize-reqPageStart, 0)
	}

	return offset, length
}
