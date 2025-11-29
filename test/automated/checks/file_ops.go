package autochecks

import (
	"os"
	"path/filepath"
)

// CheckFileOps covers create, append, mkdir, move, rename and remove.
func CheckFileOps(base string) (retErr error) {
	f := filepath.Join(base, "basic.txt")
	dir := filepath.Join(base, "test_dir")
	renamed := filepath.Join(base, "basic.renamed")
	var fh *os.File
	var err error

	_ = os.RemoveAll(f)
	_ = os.RemoveAll(renamed)
	_ = os.RemoveAll(dir)

	// create and write
	if err = os.WriteFile(f, []byte("hello world\n"), 0644); err != nil {
		retErr = err
		goto cleanup
	}

	// append
	fh, err = os.OpenFile(f, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if _, err = fh.Write([]byte("append-line\n")); err != nil {
		retErr = err
		goto cleanup
	}

	// stat
	if _, err = os.Stat(f); err != nil {
		retErr = err
		goto cleanup
	}

	// mkdir
	if err = os.Mkdir(dir, 0755); err != nil {
		retErr = err
		goto cleanup
	}

	// move into dir
	if err = os.Rename(f, filepath.Join(dir, filepath.Base(f))); err != nil {
		retErr = err
		goto cleanup
	}

	// rename out
	if err = os.Rename(filepath.Join(dir, filepath.Base(f)), renamed); err != nil {
		retErr = err
		goto cleanup
	}

	// cleanup
	if err = os.Remove(renamed); err != nil {
		retErr = err
		goto cleanup
	}
	_ = os.RemoveAll(dir)

cleanup:
	if fh != nil {
		_ = fh.Close()
	}
	_ = os.Remove(renamed)
	_ = os.RemoveAll(dir)
	_ = os.RemoveAll(f)
	return
}
