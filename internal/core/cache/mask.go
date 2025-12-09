package cache

type Mask []byte

func bitIndex(idx int) byte {
	return 1 << uint(idx&7)
}

func maskSize(size int) int {
	return (size + 7) >> 3
}

func (m *Mask) ensureSize(size int) {
	requiredBytes := maskSize(size)
	if len(*m) < requiredBytes {
		newMask := make([]byte, requiredBytes)
		copy(newMask, *m)
		*m = newMask
	}
}

func (m *Mask) setValid(start, length int) {
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
		for k := 0; k < upto; k++ {
			b |= bitIndex(bitOffset + k)
		}
		data[idx] |= b
		i += upto
	}
}

func (m Mask) isSet(i int) bool {
	if i < 0 {
		return false
	}

	byteIndex := i >> 3
	if byteIndex >= len(m) {
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
func (m Mask) shiftedRight(oldLen int, shiftedBytes int, newLen int) Mask {
	newMaskBytes := maskSize(newLen)
	newMask := make([]byte, newMaskBytes)
	if shiftedBytes == 0 || len(m) == 0 {
		// simple copy if no shift or empty mask
		copy(newMask, m)
		return newMask
	}
	for oldIdx := range oldLen {
		oldByteIndex := oldIdx >> 3
		if oldByteIndex >= len(m) {
			continue
		}
		oldBitIndex := uint(oldIdx & 7)
		if (m[oldByteIndex] & bitIndex(int(oldBitIndex))) != 0 {
			newIdx := oldIdx + shiftedBytes
			newByteIndex := newIdx >> 3
			if newByteIndex < len(newMask) {
				newMask[newByteIndex] |= bitIndex(newIdx)
			}
		}
	}
	return newMask
}
