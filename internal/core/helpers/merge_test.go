package helpers

import (
	"bytes"
	"testing"

	"github.com/mimic/internal/core/cache"
)

func TestMerge_RemoteOnly(t *testing.T) {
	remote := []byte("hello world")
	out := MergeRemoteAndBuffer(remote, 0, nil, 0, nil, 6, 5)
	if string(out) != "world" {
		t.Fatalf("remote only: expected %q got %q", "world", string(out))
	}
}

func TestMerge_BufferOnly_NilMask(t *testing.T) {
	buf := []byte("ABCDEFG")
	out := MergeRemoteAndBuffer(nil, 0, buf, 10, nil, 10, 3)
	if string(out) != "ABC" {
		t.Fatalf("buffer only nil mask: expected %q got %q", "ABC", string(out))
	}
}

func TestMerge_BufferOverridesRemote(t *testing.T) {
	remote := []byte("0123456789")
	// buffer "ABC" overlays starting at 4 -> replaces bytes 4,5,6
	buf := []byte("ABC")
	out := MergeRemoteAndBuffer(remote, 0, buf, 4, nil, 2, 6)
	// requested window 2..8 -> original remote 2,3,4,5,6,7
	// after overlay: 2,3 from remote; 4,5,6 from buf; 7 from remote
	want := []byte{'2', '3', 'A', 'B', 'C', '7'}
	if !bytes.Equal(out, want) {
		t.Fatalf("buffer overrides: expected %q got %q", string(want), string(out))
	}
}

func TestMerge_BufferMask_PageGranular(t *testing.T) {
	// Create remote and buffer spanning two pages.
	remote := bytes.Repeat([]byte{'R'}, cache.PageSize*2)
	buf := bytes.Repeat([]byte{'B'}, cache.PageSize*2)

	// Mask only marks the second page of buf as dirty.
	maskBytes := int((int64(len(buf)) + cache.PageSize) >> 12)
	m := make(cache.Mask, maskBytes)
	// mark page index 1 (second page)
	pageIdx := int64(1)
	byteIdx := int(pageIdx >> 3)
	bit := byte(1 << uint(pageIdx&7))
	m[byteIdx] |= bit

	// Request the whole two-page window
	out := MergeRemoteAndBuffer(remote, 0, buf, 0, m, 0, cache.PageSize*2)

	// First page should remain remote, second page should be from buf
	if len(out) != cache.PageSize*2 {
		t.Fatalf("unexpected out length: want %d got %d", cache.PageSize*2, len(out))
	}
	if !bytes.Equal(out[:cache.PageSize], remote[:cache.PageSize]) {
		t.Fatalf("first page should be remote")
	}
	if !bytes.Equal(out[cache.PageSize:], buf[cache.PageSize:]) {
		t.Fatalf("second page should be from buffer (dirty)")
	}
}
