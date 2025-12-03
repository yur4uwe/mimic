package autochecks

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"
)

// CheckLargeWrite writes a 10 MiB file to validate large writes.
func CheckLargeWrite(base string) (retErr error) {
	fpath := path.Join(base, "largefile")
	var info os.FileInfo
	var err error
	var retries int = 5
	var out *os.File
	zero := bytes.Repeat([]byte{0}, 1024*1024) // 1 MiB

	_ = os.RemoveAll(fpath)

	out, err = os.Create(fpath)
	if err != nil {
		retErr = fmt.Errorf("failed to create file: %w", err)
		goto cleanup
	}

	for range 10 {
		if _, err = out.Write(zero); err != nil {
			retErr = fmt.Errorf("failed to write to file: %w", err)
			goto cleanup
		}
	}
	_ = out.Close()

retry:
	info, err = os.Stat(fpath)
	if err != nil {
		retErr = fmt.Errorf("failed to stat file: %w", err)
		goto cleanup
	}
	if info.Size() < 10*1024*1024 {
		fmt.Printf("Incorrect size %d, trying again\n", info.Size())
		if retries == 0 {
			retErr = fmt.Errorf("failed to correctly stat a file after 5 tries")
			goto cleanup
		}
		retries--
		time.Sleep(500 * time.Millisecond)
		goto retry
	}

	fmt.Printf("Succeeded after %d retries\n", 5-retries+1)

cleanup:
	if out != nil {
		_ = out.Close()
	}
	_ = os.RemoveAll(fpath)
	return
}
