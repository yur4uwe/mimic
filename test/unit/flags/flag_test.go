package flags

import (
	"os"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := flags.OpenFlag(tt.val)

			// ReadAllowed / WriteAllowed semantics: tests assume POSIX-like behavior:
			// - ReadAllowed: true unless O_WRONLY is set
			// - WriteAllowed: true if O_WRONLY or O_RDWR is set
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
