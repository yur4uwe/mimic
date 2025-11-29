package autochecks

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckTruncate verifies truncating down and up works.
func CheckTruncate(base string) (retErr error) {
	fpath := filepath.Join(base, "tfile")
	var err error
	var b []byte

	if err = os.WriteFile(fpath, []byte("0123456789"), 0644); err != nil {
		retErr = err
		goto cleanup
	}
	// truncate smaller
	if err = os.Truncate(fpath, 4); err != nil {
		retErr = err
		goto cleanup
	}
	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if len(b) != 4 {
		retErr = fmt.Errorf("truncate down wrong size: %d", len(b))
		goto cleanup
	}
	// extend
	if err = os.Truncate(fpath, 10); err != nil {
		retErr = err
		goto cleanup
	}
	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if len(b) != 10 {
		retErr = fmt.Errorf("truncate up wrong size: %d", len(b))
		goto cleanup
	}

cleanup:
	_ = os.RemoveAll(fpath)
	return
}
