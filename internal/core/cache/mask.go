package cache

const (
	PageSize = 4096 // bytes = 4 kB
)

func pageIndexMask(pageIndex int64) (bitIndex byte, byteIndex int) {
	bitIndex = 1 << uint(pageIndex&7)
	byteIndex = int(pageIndex >> 3)
	return
}

type Mask []byte

func maskSize(size int64) int {
	return int((size + PageSize) >> 12)
}

func (m *Mask) ensureSize(size int64) {
	requiredBytes := maskSize(size)
	if len(*m) < requiredBytes {
		newMask := make([]byte, requiredBytes)
		copy(newMask, *m)
		*m = newMask
	}
}

func (m *Mask) smearPages(start, length int64) {
	if length <= 0 {
		return
	}
	end := start + length
	m.ensureSize(end)

	startPageIdx := start >> 12
	endPageIdx := (end + PageSize - 1) >> 12

	for pageIdx := startPageIdx; pageIdx < endPageIdx; pageIdx++ {
		pageMaskBitIndex, pageMaskByteIndex := pageIndexMask(pageIdx)
		(*m)[pageMaskByteIndex] |= pageMaskBitIndex
	}
}

func (m Mask) IsDirty(byteIndex int64) bool {
	if byteIndex < 0 {
		return false
	}

	pageMaskBitIndex, pageByteIndex := pageIndexMask(byteIndex >> 12)
	if pageByteIndex >= len(m) {
		return false
	}

	return (m[pageByteIndex] & pageMaskBitIndex) != 0
}

func (m *Mask) IsDirtyPage(pageIdx int64) bool {
	pageMaskBitIndex, pageMaskByteIndex := pageIndexMask(pageIdx)
	if pageMaskByteIndex >= len(*m) {
		return false
	}
	return ((*m)[pageMaskByteIndex] & pageMaskBitIndex) != 0
}

func (m *Mask) IsDirtyRange(start, length int64) bool {
	if length <= 0 {
		return false
	}

	startPageIdx := start >> 12
	endPageIdx := (start + length + PageSize - 1) >> 12
	for pageIdx := startPageIdx; pageIdx < endPageIdx; pageIdx++ {
		if !m.IsDirtyPage(pageIdx) {
			return false
		}
	}
	return true
}

func (m *Mask) clear() {
	*m = nil
}

// shiftedRight returns a new Mask shifted by shiftedBytes to the right,
// needed when the buffer is grown at the beginning.
// for growing at the end, use smearPages instead.
func (m Mask) shiftedRight(shiftedBytes int64, newLen int64) Mask {
	newMaskBytes := maskSize(newLen)
	newMask := make([]byte, newMaskBytes)

	// nothing to shift or empty mask
	if shiftedBytes == 0 || len(m) == 0 {
		copy(newMask, m)
		return newMask
	}

	shiftedPages := shiftedBytes >> 12
	if shiftedPages == 0 {
		// less than a page shift, copy existing mask
		copy(newMask, m)
		return newMask
	}

	if shiftedPages%8 == 0 {
		// byte-aligned shift
		byteShift := shiftedPages >> 3
		for i := 0; i < len(m); i++ {
			newIdx := i + int(byteShift)
			if newIdx >= len(newMask) {
				break
			}
			newMask[newIdx] = m[i]
		}
		return newMask
	}

	// left shift pages inside the byte array, preserve carry
	// for larger shifts subtract multiples of 8 and use newMask[i + byteShift]
	byteOffset := shiftedPages >> 3
	shiftedPages = shiftedPages & 7
	byteCarry := byte(0)
	for i := range m {
		currentByteState := m[i]
		newByteState := (currentByteState << uint(shiftedPages)) | byteCarry
		newMask[i+int(byteOffset)] = newByteState
		// prepare carry for next byte
		byteCarry = (currentByteState >> uint(8-shiftedPages)) & 0xFF
	}

	return newMask
}
