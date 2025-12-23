package cache

import (
	"testing"
)

func TestMaskEnsureSize(t *testing.T) {
	var m Mask
	// addressable value; pointer receiver can be called on it
	m.ensureSize(16)
	want := maskSize(16)
	if len(m) != want {
		t.Fatalf("ensureSize: expected mask bytes %d, got %d", want, len(m))
	}
}

func TestSmearPagesAndIsDirty(t *testing.T) {
	var m Mask
	// smear a small byte range that lives within the first page
	m.smearPages(3, 5) // bytes 3..7 -> page 0

	if !m.IsDirty(0) {
		t.Fatalf("expected page containing byte 0 to be dirty")
	}
	if !m.IsDirty(3) {
		t.Fatalf("expected page containing byte 3 to be dirty")
	}
	if !m.IsDirty(PageSize - 1) {
		t.Fatalf("expected last byte of first page to be dirty")
	}
	if m.IsDirty(PageSize) {
		t.Fatalf("expected first byte of second page to be clean")
	}
}

func TestSmearPagesCrossPageBoundary(t *testing.T) {
	var m Mask
	// range that crosses from page 0 into page 1
	start := int64(PageSize - 100)
	length := int64(200)
	m.smearPages(start, length)

	if !m.IsDirty(start) {
		t.Fatalf("expected byte %d to be dirty", start)
	}
	if !m.IsDirty(PageSize + 50) {
		t.Fatalf("expected byte %d in next page to be dirty", PageSize+50)
	}
}

func TestMaskClear(t *testing.T) {
	var m Mask
	m.smearPages(0, PageSize) // mark first page
	if !m.IsDirty(PageSize / 2) {
		t.Fatalf("precondition failed: expected page to be dirty")
	}
	m.clear()
	if m.IsDirty(PageSize / 2) {
		t.Fatalf("clear: expected page to be unset")
	}
}

func TestShiftedRight_PageShift(t *testing.T) {
	var m Mask
	// mark page 0, page 1, and page 5
	m.smearPages(0, PageSize*2)
	m.smearPages(PageSize*5, PageSize)

	oldLen := int64(PageSize * 6)
	shiftBy := int64(PageSize * 3)
	newLen := int64(oldLen + shiftBy)
	newMask := m.shiftedRight(shiftBy, newLen)

	// old page 0 -> new at offset shiftBy
	if !newMask.IsDirty(0 + shiftBy) {
		t.Fatalf("shiftedRight: expected old page 0 to move to %d", shiftBy)
	}
	// old page 1 -> new at shiftBy + PageSize
	if !newMask.IsDirty(PageSize + shiftBy) {
		t.Fatalf("shiftedRight: expected old page 1 to move to %d", PageSize+shiftBy)
	}
	// old page 5 -> new at shiftBy + 5*PageSize
	if !newMask.IsDirty(5*PageSize + shiftBy) {
		t.Fatalf("shiftedRight: expected old page 5 to move to %d", 5*PageSize+shiftBy)
	}
}

func TestShiftedRight_ByteAlignedShift(t *testing.T) {
	var m Mask
	// mark page 0, page 1, and page 5
	m.smearPages(0, PageSize*2)
	m.smearPages(PageSize*5, PageSize)

	oldLen := int64(PageSize * 6)
	shiftedPages := int64(8) // byte-aligned shift (8 pages -> 1 mask byte)
	shiftBy := int64(PageSize * shiftedPages)
	newLen := int64(oldLen + shiftBy)
	newMask := m.shiftedRight(shiftBy, newLen)

	// old page 0 -> new at offset shiftBy
	if !newMask.IsDirty(0 + shiftBy) {
		t.Fatalf("shiftedRight (byte-aligned): expected old page 0 to move to %d", shiftBy)
	}
	// old page 1 -> new at shiftBy + PageSize
	if !newMask.IsDirty(PageSize + shiftBy) {
		t.Fatalf("shiftedRight (byte-aligned): expected old page 1 to move to %d", PageSize+shiftBy)
	}
	// old page 5 -> new at shiftBy + 5*PageSize
	if !newMask.IsDirty(5*PageSize + shiftBy) {
		t.Fatalf("shiftedRight (byte-aligned): expected old page 5 to move to %d", 5*PageSize+shiftBy)
	}
}
