package cache

import (
	"testing"
)

func TestFileBuffer_WriteAppendRead(t *testing.T) {
	var fb FileBuffer

	if err := fb.WriteAt(0, []byte("BASE")); err != nil {
		t.Fatalf("WriteAt(0,BASE) failed: %v", err)
	}
	if err := fb.WriteAt(4, []byte("A")); err != nil {
		t.Fatalf("WriteAt(4,A) failed: %v", err)
	}

	cp := fb.CopyBuffer()
	if fb.Base != 0 {
		t.Fatalf("CopyBuffer base: want 0 got %d", fb.Base)
	}
	if string(cp.Data) != "BASEA" {
		t.Fatalf("buffer content: want %q got %q", "BASEA", string(cp.Data))
	}

	// Validate mask bits
	for i := int64(0); i < 5; i++ {
		if !fb.IsValidAt(i) {
			t.Fatalf("expected valid bit at %d", i)
		}
	}

	// ReadAt should return the full content
	out, err := fb.ReadAt(0, 5)
	if err != nil {
		t.Fatalf("ReadAt error: %v", err)
	}
	if string(out) != "BASEA" {
		t.Fatalf("ReadAt content: want %q got %q", "BASEA", string(out))
	}
}

func TestFileBuffer_PrependBehavior(t *testing.T) {
	var fb FileBuffer

	if err := fb.WriteAt(4, []byte("X")); err != nil {
		t.Fatalf("initial WriteAt failed: %v", err)
	}

	if fb.Base != 4 {
		t.Fatalf("initial base: want 4 got %d", fb.Base)
	}
	if len(fb.Data) != 1 || fb.Data[0] != 'X' {
		t.Fatalf("initial data: want %q got %q", "X", string(fb.Data))
	}

	if err := fb.WriteAt(0, []byte("BASE")); err != nil {
		t.Fatalf("prepend WriteAt failed: %v", err)
	}

	cp := fb.CopyBuffer()
	if fb.Base != 0 {
		t.Fatalf("after prepend base: want 0 got %d", fb.Base)
	}
	if string(cp.Data) != "BASEX" {
		t.Fatalf("after prepend content: want %q got %q", "BASEX", string(cp.Data))
	}

	// mask: indices 0..4 should be valid (0..3 from BASE, 4 from X)
	for i := int64(0); i <= 4; i++ {
		if !fb.IsValidAt(i) {
			t.Fatalf("expected valid bit at %d after prepend", i)
		}
	}
}

func TestFileBuffer_OverwriteWithin(t *testing.T) {
	var fb FileBuffer

	if err := fb.WriteAt(0, []byte("HELLO")); err != nil {
		t.Fatalf("WriteAt HELLO failed: %v", err)
	}
	// overwrite 'E' with 'i' at offset 1
	if err := fb.WriteAt(1, []byte("i")); err != nil {
		t.Fatalf("WriteAt overwrite failed: %v", err)
	}
	cp := fb.CopyBuffer()
	if string(cp.Data) != "HiLLO" {
		t.Fatalf("overwrite result: want %q got %q", "HiLLO", string(cp.Data))
	}
	// mask should mark first 4 bytes valid
	for i := range int64(5) {
		if !fb.IsValidAt(i) {
			t.Fatalf("expected valid bit at %d after overwrite", i)
		}
	}
}

func TestFileBuffer_PartialMaskGap(t *testing.T) {
	var fb FileBuffer

	// create a small sparse buffer: write at offsets 5 and 7
	if err := fb.WriteAt(5, []byte("A")); err != nil {
		t.Fatalf("WriteAt(5,A) failed: %v", err)
	}
	if err := fb.WriteAt(7, []byte("B")); err != nil {
		t.Fatalf("WriteAt(7,B) failed: %v", err)
	}

	cp := fb.CopyBuffer()
	if fb.Base != 5 {
		t.Fatalf("sparse base: want 5 got %d", fb.Base)
	}
	// expected internal data layout: indices 0:'A',1:0,2:'B' -> length 3
	if len(cp.Data) != 3 {
		t.Fatalf("sparse data length: want 3 got %d", len(cp.Data))
	}
	if cp.Data[0] != 'A' || cp.Data[1] != 0 || cp.Data[2] != 'B' {
		t.Fatalf("sparse data content: want [%q,0,%q] got [%q,%d,%q]",
			'A', 'B', cp.Data[0], cp.Data[1], cp.Data[2])
	}

	// mask: only offsets 0 and 2 (absolute 5 and 7) should be valid
	// mask: with page-based mask implementation bytes within the same page
	// (absolute offsets base..base+2) will all be considered valid
	if !fb.IsValidAt(fb.Base+0) || !fb.IsValidAt(fb.Base+1) || !fb.IsValidAt(fb.Base+2) {
		t.Fatalf("sparse mask (page-granular): expected [true,true,true], got [%v,%v,%v]",
			fb.IsValidAt(fb.Base+0), fb.IsValidAt(fb.Base+1), fb.IsValidAt(fb.Base+2))
	}
}

func TestFileBuffer_ClearAndBounds(t *testing.T) {
	var fb FileBuffer

	if err := fb.WriteAt(0, []byte("DATA")); err != nil {
		t.Fatalf("WriteAt DATA failed: %v", err)
	}
	// ReadAt with out-of-bounds should return ErrOutOfBounds
	if _, err := fb.ReadAt(3, 5); err == nil {
		t.Fatalf("expected ReadAt out-of-bounds error, got nil")
	}

	// Clear
	fb.Clear()
	if fb.Size() != 0 {
		t.Fatalf("after Clear size: want 0 got %d", fb.Size())
	}
	if fb.IsValidAt(0) {
		t.Fatalf("after Clear expected no valid bits")
	}
	if buf := fb.CopyBuffer(); buf != nil {
		t.Fatalf("after Clear expected nil buffer copy, got len %d", len(buf.Data))
	}
}
