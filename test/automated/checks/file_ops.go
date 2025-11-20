package autochecks

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckFileOps covers create, append, mkdir, move, rename and remove.
func CheckFileOps(base string) error {
	f := join(base, "basic.txt")
	dir := join(base, "test_dir")
	renamed := join(base, "basic.renamed")

	ensureAbsent(f)
	ensureAbsent(renamed)
	_ = os.RemoveAll(dir)

	// create and write
	if err := writeFile(f, []byte("hello world\n")); err != nil {
		return err
	}

	// append
	fh, err := os.OpenFile(f, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	if _, err := fh.Write([]byte("append-line\n")); err != nil {
		_ = fh.Close()
		return err
	}
	_ = fh.Close()

	// stat
	if _, err := os.Stat(f); err != nil {
		return err
	}

	// mkdir
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	// move into dir
	if err := os.Rename(f, filepath.Join(dir, filepath.Base(f))); err != nil {
		return err
	}

	// rename out
	if err := os.Rename(filepath.Join(dir, filepath.Base(f)), renamed); err != nil {
		return err
	}

	// cleanup
	if err := os.Remove(renamed); err != nil {
		return err
	}
	_ = os.RemoveAll(dir)

	ensureAbsent(f)
	fmt.Println("file ops OK")
	return nil
}
