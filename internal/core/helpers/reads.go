package helpers

import (
	"fmt"

	"github.com/mimic/internal/core/cache"
)

const (
	READAHEAD_DEFAULT int64 = 64 * 1024 // 64 KB mimimum readahead
)

func PageAlignedRange(offset, length, remoteSize int64) (int64, int64) {
	reqPageStart := offset - (offset % cache.PageSize)
	reqPagesCount := (length + (offset - reqPageStart) + cache.PageSize - 1) / cache.PageSize
	readAheadLen := reqPagesCount * cache.PageSize

	// compute readahead length: at least requested length, at least READAHEAD_DEFAULT,
	// but capped to READAHEAD_MAX and remote size when known
	readAheadLen = max(readAheadLen, READAHEAD_DEFAULT)

	// don't read past known remote end
	if reqPageStart+readAheadLen > remoteSize {
		fmt.Println("EOF")
		readAheadLen = max(remoteSize-reqPageStart, 0)
	}

	return reqPageStart, readAheadLen
}
