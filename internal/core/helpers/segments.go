package helpers

import "math"

// MergeSegmentsInto merges the provided in-memory segments into the base slice.
// segments keys are offsets (int64) and values are bytes to copy at that offset.
func MergeSegmentsInto(base []byte, segments map[int64][]byte) []byte {
	// snapshot to avoid iterating a map that might be mutated concurrently
	type entry struct {
		off int64
		buf []byte
	}
	ents := make([]entry, 0, len(segments))
	for off, b := range segments {
		if len(b) == 0 || off < 0 {
			continue
		}
		ents = append(ents, entry{off: off, buf: b})
	}

	// compute required length (with validation)
	var maxEnd int64 = int64(len(base))
	for _, e := range ents {
		end := e.off + int64(len(e.buf))
		if end > maxEnd {
			maxEnd = end
		}
	}
	if maxEnd > int64(math.MaxInt) {
		maxEnd = int64(math.MaxInt)
	}

	merged := make([]byte, int(maxEnd))
	copy(merged, base)

	// apply segments with bounds checks and safe int conversion
	for _, e := range ents {
		off := int(e.off)
		if off >= len(merged) {
			continue
		}
		n := len(e.buf)
		if n > len(merged)-off {
			n = len(merged) - off
		}
		copy(merged[off:off+n], e.buf[:n])
	}
	return merged
}

func AddSegment(segments map[int64][]byte, offset int64, data []byte) map[int64][]byte {
	if segments == nil {
		segments = make(map[int64][]byte)
	}

	segments[offset] = data
	return segments
}

// AddToBuffer merges 'data' into 'buffer' at given offset.
// buffer is treated as a zero-based file-image; the function will grow
// buffer if needed and copy data into it. Returns the updated buffer.
func AddToBuffer(buffer []byte, offset int64, data []byte) []byte {
	if len(data) == 0 {
		return buffer
	}
	if offset < 0 {
		// defensive: treat negative offset as zero
		offset = 0
	}

	end64 := offset + int64(len(data))
	if end64 > int64(^uint(0)>>1) {
		// too large, avoid panic
		return buffer
	}
	end := int(end64)

	if end > len(buffer) {
		// grow buffer to accomodate new data
		newBuf := make([]byte, end)
		copy(newBuf, buffer)
		buffer = newBuf
	}

	start := int(offset)
	copy(buffer[start:end], data)
	return buffer
}

// MergeBufferIntoBase overlays buffer onto base (buffer anchored at offset 0).
// The returned slice length is max(len(base), len(buffer)).
func MergeBufferIntoBase(base []byte, buffer []byte) []byte {
	if len(buffer) == 0 {
		// nothing to overlay
		out := make([]byte, len(base))
		copy(out, base)
		return out
	}
	maxLen := len(base)
	if len(buffer) > maxLen {
		maxLen = len(buffer)
	}
	merged := make([]byte, maxLen)
	copy(merged, base)
	copy(merged[:len(buffer)], buffer)
	return merged
}

// MergeRemoteAndBuffer merges a remote slice (which represents bytes starting at
// remoteStart) and an in-memory buffer (bufData starting at bufStart) into the
// requested window [reqStart, reqStart+reqLen). Bytes from bufData override
// remote when they overlap. The returned slice length is up to reqLen and may
// be shorter (EOF semantics).
func MergeRemoteAndBuffer(remote []byte, remoteStart int64, bufData []byte, bufStart int64, reqStart int64, reqLen int) []byte {
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
		start := bufStart
		if start < reqStart {
			start = reqStart
		}
		end := bufEnd
		if end > reqEnd {
			end = reqEnd
		}
		if end > start {
			dst := int(start - reqStart)
			src := int(start - bufStart)
			copy(out[dst:dst+int(end-start)], bufData[src:src+int(end-start)])
		}
	}

	return out
}
