package autochecks

import (
	"fmt"
	"path/filepath"
)

// CheckTruncate verifies truncating down and up works.
func CheckTruncate(base string) (retErr error) {
	fpath := filepath.Join(base, "tfile")
	var err error
	var b []byte

	if err = writeFile(fpath, []byte("0123456789")); err != nil {
		retErr = err
		goto cleanup
	}
	// truncate smaller
	if err = truncateFile(fpath, 4); err != nil {
		retErr = err
		goto cleanup
	}
	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if len(b) != 4 {
		retErr = fmt.Errorf("truncate down wrong size: %d", len(b))
		goto cleanup
	}
	// extend
	if err = truncateFile(fpath, 10); err != nil {
		retErr = err
		goto cleanup
	}
	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if len(b) != 10 {
		retErr = fmt.Errorf("truncate up wrong size: %d", len(b))
		goto cleanup
	}

cleanup:
	ensureAbsent(fpath)
	return
}
