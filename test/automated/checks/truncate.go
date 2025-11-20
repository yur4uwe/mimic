package autochecks

import (
	"fmt"
)

// CheckTruncate verifies truncating down and up works.
func CheckTruncate(base string) error {
	fpath := join(base, "tfile")
	if err := writeFile(fpath, []byte("0123456789")); err != nil {
		return err
	}
	// truncate smaller
	if err := truncateFile(fpath, 4); err != nil {
		return err
	}
	b, err := readAll(fpath)
	if err != nil {
		return err
	}
	if len(b) != 4 {
		return fmt.Errorf("truncate down wrong size: %d", len(b))
	}
	// extend
	if err := truncateFile(fpath, 10); err != nil {
		return err
	}
	b, err = readAll(fpath)
	if err != nil {
		return err
	}
	if len(b) != 10 {
		return fmt.Errorf("truncate up wrong size: %d", len(b))
	}
	ensureAbsent(fpath)
	return nil
}
