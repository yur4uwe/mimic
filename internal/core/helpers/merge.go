package helpers

// MergeSegmentsInto merges the provided in-memory segments into the base slice.
// segments map keys are offsets (int64) and values are the bytes to copy at that offset.
// The result length is extended if any segment goes past the base length.
func MergeSegmentsInto(base []byte, segments map[int64][]byte) []byte {
	maxEnd := int64(len(base))
	for off, seg := range segments {
		if seg == nil {
			continue
		}
		end := off + int64(len(seg))
		if end > maxEnd {
			maxEnd = end
		}
	}

	merged := make([]byte, maxEnd)
	copy(merged, base)

	for off, seg := range segments {
		if seg == nil {
			continue
		}
		copy(merged[off:], seg)
	}

	return merged
}
