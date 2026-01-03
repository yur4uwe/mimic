package fs

import (
	"testing"

	"github.com/mimic/internal/core/flags"
)

func TestPageAlignedRange(t *testing.T) {
	tests := []struct {
		name       string
		InOffset   int64
		InLength   int64
		OutOffset  int64
		OutLength  int64
		RemoteSize int64
	}{
		{
			name:       "Non-aligned read, very small request",
			InOffset:   4,
			InLength:   1,
			RemoteSize: 4,
			OutOffset:  0,
			OutLength:  64 * 1024, // READAHEAD_DEFAULT minimum (64 KiB)
		},
		{
			name:       "Aligned read, remote large",
			InOffset:   8192,
			InLength:   4096,
			RemoteSize: 100000,
			OutOffset:  8192,
			OutLength:  64 * 1024, // READAHEAD_DEFAULT minimum (64 KiB)
		},
		{
			name:       "Unaligned read, remote large",
			InOffset:   5000,
			InLength:   2000,
			RemoteSize: 100000,
			OutOffset:  4096,
			OutLength:  64 * 1024, // readahead to default (64 KiB)
		},
		{
			name:       "Read near remote end",
			InOffset:   9000,
			InLength:   2000,
			RemoteSize: 10000,
			OutOffset:  8192,
			OutLength:  1808, // clipped to remoteSize - reqPageStart
		},
		{
			name:       "Remote smaller than page start",
			InOffset:   5000,
			InLength:   1000,
			RemoteSize: 3000,
			OutOffset:  4096,
			OutLength:  0, // nothing to read
		},
		{
			name:       "Zero length read",
			InOffset:   5000,
			InLength:   0,
			RemoteSize: 100000,
			OutOffset:  4096,
			OutLength:  64 * 1024, // still at least READAHEAD_DEFAULT (64 KiB)
		},
		{
			name:       "Strange reads (kernel cache?)",
			InOffset:   1024,
			InLength:   512,
			OutOffset:  0,
			OutLength:  64 * 1024, // READAHEAD_DEFAULT minimum (64 KiB)
			RemoteSize: 100000,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fh := NewFilehandle("/tmp/f", flags.OpenFlag(0), nil)
			fh.remoteSize = tt.RemoteSize
			outOffset, outLength := fh.PageAlignedRange(tt.InOffset, tt.InLength)
			if outOffset != tt.OutOffset {
				t.Fatalf("%s: expected OutOffset %d, got %d", tt.name, tt.OutOffset, outOffset)
			}
			if outLength != tt.OutLength {
				t.Fatalf("%s: expected OutLength %d, got %d", tt.name, tt.OutLength, outLength)
			}
		})
	}
}

func TestPageAlignedRangeGrowth(t *testing.T) {
	fh := NewFilehandle("/tmp/f", flags.OpenFlag(0), nil)
	// large remote to avoid clipping
	fh.remoteSize = 10 * 1024 * 1024

	// Simulate a previous large read to trigger the growth branch.
	fh.lastReadSize = 128 * 1024 // 128 KiB

	_, outLen := fh.PageAlignedRange(0, 4096)
	want := int64(128*1024) * 2
	if outLen != want {
		t.Fatalf("growth: expected OutLength %d, got %d", want, outLen)
	}
}
