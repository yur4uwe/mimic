package helpers

import "testing"

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			outOffset, outLength := PageAlignedRange(tt.InOffset, tt.InLength, tt.RemoteSize)
			if outOffset != tt.OutOffset {
				t.Fatalf("%s: expected OutOffset %d, got %d", tt.name, tt.OutOffset, outOffset)
			}
			if outLength != tt.OutLength {
				t.Fatalf("%s: expected OutLength %d, got %d", tt.name, tt.OutLength, outLength)
			}
		})
	}
}
