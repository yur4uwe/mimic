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
	fpath := path.Join(base, "largefile.largewrite")
	var info os.FileInfo
	var err error
	var retries int = 5
	var out *os.File
	zero := bytes.Repeat([]byte{0}, 1024*1024) // 1 MiB

	_ = os.RemoveAll(fpath)

	// create file
	s := time.Now()
	out, err = os.Create(fpath)
	if err != nil {
		retErr = fmt.Errorf("failed to create file: %w", err)
		summarizeSub("Large write", "create", retErr, time.Since(s))
		goto cleanup
	}
	summarizeSub("Large write", "create", nil, time.Since(s))

	// write 10 MiB
	s = time.Now()
	for i := 0; i < 10; i++ {
		if _, err = out.Write(zero); err != nil {
			retErr = fmt.Errorf("failed to write to file: %w", err)
			summarizeSub("Large write", "writeChunks", retErr, time.Since(s))
			goto cleanup
		}
	}
	_ = out.Close()
	summarizeSub("Large write", "writeChunks", nil, time.Since(s))

retryCheck:
	// stat check (may need retries)
	s = time.Now()
	info, err = os.Stat(fpath)
	if err != nil {
		retErr = fmt.Errorf("failed to stat file: %w", err)
		summarizeSub("Large write", "stat", retErr, time.Since(s))
		goto cleanup
	}
	if info.Size() < 10*1024*1024 {
		fmt.Printf("Incorrect size %d, trying again\n", info.Size())
		if retries == 0 {
			retErr = fmt.Errorf("failed to correctly stat a file after 5 tries")
			summarizeSub("Large write", "stat", retErr, time.Since(s))
			goto cleanup
		}
		retries--
		time.Sleep(500 * time.Millisecond)
		goto retryCheck
	}
	summarizeSub("Large write", "stat", nil, time.Since(s))

	fmt.Printf("Succeeded after %d retries\n", 5-retries+1)

cleanup:
	if out != nil {
		_ = out.Close()
	}
	_ = os.RemoveAll(fpath)
	return
}
