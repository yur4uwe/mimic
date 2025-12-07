package flags

import (
	"os"
	"strings"
	"testing"

	"github.com/mimic/internal/core/flags"
)

func TestOpenFlagBasics(t *testing.T) {
	tests := []struct {
		name          string
		val           uint32
		wantRead      bool
		wantWrite     bool
		wantAppend    bool
		wantCreate    bool
		wantTruncate  bool
		wantExclusive bool
	}{
		{
			name:          "read only",
			val:           uint32(os.O_RDONLY),
			wantRead:      true,
			wantWrite:     false,
			wantAppend:    false,
			wantCreate:    false,
			wantTruncate:  false,
			wantExclusive: false,
		},
		{
			name:          "write only (create|write|truncate)",
			val:           uint32(os.O_WRONLY | os.O_CREATE | os.O_TRUNC),
			wantRead:      false,
			wantWrite:     true,
			wantAppend:    false,
			wantCreate:    true,
			wantTruncate:  true,
			wantExclusive: false,
		},
		{
			name:          "read/write append",
			val:           uint32(os.O_RDWR | os.O_APPEND),
			wantRead:      true,
			wantWrite:     true,
			wantAppend:    true,
			wantCreate:    false,
			wantTruncate:  false,
			wantExclusive: false,
		},
		{
			name:          "create exclusive",
			val:           uint32(os.O_CREATE | os.O_EXCL),
			wantRead:      true,  // no explicit write-only, so read allowed by default
			wantWrite:     false, // no write bit set
			wantAppend:    false,
			wantCreate:    true,
			wantTruncate:  false,
			wantExclusive: true,
		},
		{
			name:          "no flags",
			val:           0,
			wantRead:      true, // default behavior
			wantWrite:     false,
			wantAppend:    false,
			wantCreate:    false,
			wantTruncate:  false,
			wantExclusive: false,
		},
		{
			name:          "read/write create truncate",
			val:           uint32(os.O_RDWR | os.O_CREATE | os.O_TRUNC),
			wantRead:      true,
			wantWrite:     true,
			wantAppend:    false,
			wantCreate:    true,
			wantTruncate:  true,
			wantExclusive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := flags.OpenFlag(tt.val)

			gotRead := f.ReadAllowed()
			gotWrite := f.WriteAllowed()

			if gotRead != tt.wantRead {
				t.Fatalf("ReadAllowed: flag=%#x got=%v want=%v", tt.val, gotRead, tt.wantRead)
			}
			if gotWrite != tt.wantWrite {
				t.Fatalf("WriteAllowed: flag=%#x got=%v want=%v", tt.val, gotWrite, tt.wantWrite)
			}

			if f.Append() != tt.wantAppend {
				t.Fatalf("Append bit: flag=%#x got=%v want=%v", tt.val, f.Append(), tt.wantAppend)
			}

			if f.Create() != tt.wantCreate {
				t.Fatalf("Create: flag=%#x got=%v want=%v", tt.val, f.Create(), tt.wantCreate)
			}
			if f.Truncate() != tt.wantTruncate {
				t.Fatalf("Truncate: flag=%#x got=%v want=%v", tt.val, f.Truncate(), tt.wantTruncate)
			}
			if f.Exclusive() != tt.wantExclusive {
				t.Fatalf("Exclusive: flag=%#x got=%v want=%v", tt.val, f.Exclusive(), tt.wantExclusive)
			}
		})
	}
}

func TestOpenFlagString(t *testing.T) {
	tests := []struct {
		name string
		val  uint32
		want string
	}{
		{"read only", uint32(os.O_RDONLY), "O_RDONLY"},
		{"write only", uint32(os.O_WRONLY), "O_WRONLY"},
		{"read/write", uint32(os.O_RDWR), "O_RDWR"},
		{"create", uint32(os.O_CREATE), "O_CREATE"},
		{"truncate", uint32(os.O_TRUNC), "O_TRUNC"},
		{"append", uint32(os.O_APPEND), "O_APPEND"},
		{"exclusive", uint32(os.O_EXCL), "O_EXCL"},
		{"read/write create truncate", uint32(os.O_RDWR | os.O_CREATE | os.O_TRUNC), "O_RDWR|O_CREATE|O_TRUNC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := flags.OpenFlag(tt.val)
			got := f.String()
			if !strings.Contains(got, tt.want) {
				t.Fatalf("String: flag=%#x got=%q want=%q", tt.val, got, tt.want)
			}
		})
	}
}
