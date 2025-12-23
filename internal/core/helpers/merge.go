package helpers

import "github.com/mimic/internal/core/cache"

// MergeRemoteAndBuffer merges a remote slice (which represents bytes starting at
// remoteStart) and an in-memory buffer (bufData starting at bufStart) into the
// requested window [reqStart, reqStart+reqLen). Bytes from bufData override
// remote when they overlap. The returned slice length is up to reqLen and may
// be shorter (EOF semantics).
func MergeRemoteAndBuffer(remote []byte, remoteStart int64, bufData []byte, bufStart int64, bufMask cache.Mask, reqStart int64, reqLen int) []byte {
	reqEnd := reqStart + int64(reqLen)

	remoteLen := int64(len(remote))
	remoteEnd := remoteStart + remoteLen

	bufLen := int64(len(bufData))
	bufEnd := bufStart + bufLen

	// Determine the merged coverage within the requested window
	maxEnd := min(max(bufEnd, max(remoteEnd, reqStart)), reqEnd)
	if maxEnd <= reqStart {
		return []byte{}
	}

	outLen := int(maxEnd - reqStart)
	out := make([]byte, outLen)

	// Copy any overlapping remote data into out
	if remoteLen > 0 {
		start := max(remoteStart, reqStart)
		end := min(remoteEnd, reqEnd)
		if end > start {
			dst := int(start - reqStart)
			src := int(start - remoteStart)
			copy(out[dst:dst+int(end-start)], remote[src:src+int(end-start)])
		}
	}

	// Overlay buffer data (buffer dominates remote)
	if bufLen > 0 {
		start := max(bufStart, reqStart)
		end := min(bufEnd, reqEnd)
		if end > start {
			dst := start - reqStart
			src := start - bufStart
			if bufMask == nil {
				copy(out[dst:dst+end-start], bufData[src:src+end-start])
			} else {
				for i := int64(0); i < end-start; i++ {
					if bufMask.IsDirty(src + i) {
						out[dst+i] = bufData[src+i]
					}
				}
			}
		}
	}

	return out
}
