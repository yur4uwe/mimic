package cache

type Mask []byte

func bitIndex(idx int64) byte {
	return 1 << uint(idx&7)
}

func maskSize(size int64) int {
	return int((size + 7) >> 3)
}

func (m *Mask) ensureSize(size int64) {
	requiredBytes := maskSize(size)
	if len(*m) < requiredBytes {
		newMask := make([]byte, requiredBytes)
		copy(newMask, *m)
		*m = newMask
	}
}

func (m *Mask) setValid(start, length int64) {
	if length <= 0 {
		return
	}
	end := start + length
	m.ensureSize(end)
	data := *m
	for i := start; i < end; {
		idx := i >> 3
		bitOffset := i & 7
		upto := ((idx + 1) << 3) - i
		if upto > (end - i) {
			upto = end - i
		}
		var b byte
		for k := int64(0); k < upto; k++ {
			b |= bitIndex(bitOffset + k)
		}
		data[idx] |= b
		i += upto
	}
}

func (m Mask) IsSet(i int64) bool {
	if i < 0 {
		return false
	}

	byteIndex := i >> 3
	if byteIndex >= int64(len(m)) {
		return false
	}

	return (m[byteIndex] & bitIndex(i)) != 0
}

func (m *Mask) clear() {
	*m = nil
}

// shiftedRight returns a new Mask shifted by shiftedBytes to the right,
// needed when the buffer is grown at the beginning.
// for growing at the end, use setValid instead.
func (m Mask) shiftedRight(oldLen int64, shiftedBytes int64, newLen int64) Mask {
	newMaskBytes := maskSize(newLen)
	newMask := make([]byte, newMaskBytes)

	// fast-path: nothing to shift or empty mask
	if shiftedBytes == 0 || len(m) == 0 {
		copy(newMask, m)
		return newMask
	}

	for oldIdx := range oldLen {
		if !m.IsSet(oldIdx) {
			continue
		}

		newIdx := oldIdx + shiftedBytes
		if newIdx < 0 || newIdx >= newLen {
			// shifted bit would be out of range for the new mask
			continue
		}
		newByteIndex := newIdx >> 3
		newMask[newByteIndex] |= bitIndex(newIdx)
	}

	return newMask
}
