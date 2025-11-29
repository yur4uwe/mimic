package autochecks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// CheckLargeWrite writes a 10 MiB file to validate large writes.
func CheckLargeWrite(base string) (retErr error) {
	fpath := filepath.Join(base, "largefile")
	var info os.FileInfo
	var err error
	zero := bytes.Repeat([]byte{0}, 1024*1024) // 1 MiB
	var out *os.File

	_ = os.RemoveAll(fpath)

	out, err = os.Create(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}

	for i := 0; i < 10; i++ {
		if _, err = out.Write(zero); err != nil {
			retErr = err
			goto cleanup
		}
	}
	_ = out.Close()

	info, err = os.Stat(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if info.Size() < 10*1024*1024 {
		retErr = fmt.Errorf("big file size too small: %d", info.Size())
		goto cleanup
	}

cleanup:
	if out != nil {
		_ = out.Close()
	}
	_ = os.RemoveAll(fpath)
	return
}
