package autochecks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CheckLargeWrite writes a 10 MiB file to validate large writes.
func CheckLargeWrite(base string) error {
	fpath := filepath.Join(base, "big.bin")
	ensureAbsent(fpath)

	out, err := os.Create(fpath)
	if err != nil {
		return err
	}

	zero := bytes.Repeat([]byte{0}, 1024*1024) // 1 MiB
	for i := 0; i < 10; i++ {
		if _, err := out.Write(zero); err != nil {
			out.Close()
			return err
		}
	}
	out.Close()

	// untangle race conditions
	time.Sleep(5 * time.Second)

	info, err := os.Stat(fpath)
	if err != nil {
		return err
	}
	if info.Size() < 10*1024*1024 {
		return fmt.Errorf("big file size too small: %d", info.Size())
	}
	ensureAbsent(fpath)
	return nil
}
