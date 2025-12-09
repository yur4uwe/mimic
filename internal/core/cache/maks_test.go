package cache

import "testing"

func TestMaskEnsureSize(t *testing.T) {
	var m Mask
	// addressable value; pointer receiver can be called on it
	m.ensureSize(16)
	want := maskSize(16)
	if len(m) != want {
		t.Fatalf("ensureSize: expected mask bytes %d, got %d", want, len(m))
	}
}

func TestMaskSetValidAndIsSet(t *testing.T) {
	var m Mask
	// set bytes 3..7 (length 5 -> indices 3,4,5,6,7)
	m.setValid(3, 5)

	cases := map[int]bool{
		2: false,
		3: true,
		4: true,
		7: true,
		8: false,
	}
	for idx, exp := range cases {
		if got := m.isSet(idx); got != exp {
			t.Fatalf("isSet(%d): expected %v, got %v", idx, exp, got)
		}
	}
}

func TestMaskClear(t *testing.T) {
	var m Mask
	m.setValid(0, 4) // set 0..3
	if !m.isSet(2) {
		t.Fatalf("precondition failed: expected bit 2 set")
	}
	m.clear()
	for i := range 8 {
		if m.isSet(i) {
			t.Fatalf("clear: expected bit %d to be unset", i)
		}
	}
}

func TestMaskShiftedRight(t *testing.T) {
	var m Mask
	// mark bytes 0..1 and byte 5
	m.setValid(0, 2) // indices 0,1
	m.setValid(5, 1) // index 5

	oldLen := 6
	shiftBy := 3
	newLen := oldLen + shiftBy
	newMask := m.shiftedRight(oldLen, shiftBy, newLen)

	// old 0 -> new 3, old1 -> new4, old5 -> new8
	expectSet := map[int]bool{
		3: true,
		4: true,
		8: true,
		0: false,
		1: false,
		5: false,
	}
	for idx, exp := range expectSet {
		if got := newMask.isSet(idx); got != exp {
			t.Fatalf("shiftedRight: bit %d expected %v got %v", idx, exp, got)
		}
	}
}

func TestMaskSetValidCrossByteBoundary(t *testing.T) {
	var m Mask
	// set a range that crosses a mask byte boundary, e.g., start=6 length=6 -> indices 6..11
	m.setValid(6, 6)
	for i := range 12 {
		expect := (i >= 6 && i < 12)
		if got := m.isSet(i); got != expect {
			t.Fatalf("cross-boundary setValid: index %d expected %v got %v", i, expect, got)
		}
	}
}
