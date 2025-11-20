package autochecks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// CheckConcurrentAppendRead performs concurrent appends and a reader tailing the file.
func CheckConcurrentAppendRead(base string) error {
	fpath := join(base, "stream.txt")
	ensureAbsent(fpath)
	if err := writeFile(fpath, []byte{}); err != nil {
		return err
	}

	const n = 40
	var wg sync.WaitGroup
	wg.Add(2)

	var appendErr error
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				appendErr = err
				return
			}
			if _, err := f.Write([]byte(fmt.Sprintf("x%02d\n", i))); err != nil {
				_ = f.Close()
				appendErr = err
				return
			}
			_ = f.Close()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	var readErr error
	go func() {
		defer wg.Done()
		seen := 0
		var last int64 = 0
		for seen < n && readErr == nil {
			info, err := os.Stat(fpath)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if info.Size() > last {
				f, err := os.Open(fpath)
				if err == nil {
					_, _ = f.Seek(last, io.SeekStart)
					buf := make([]byte, info.Size()-last)
					_, _ = io.ReadFull(f, buf)
					_ = f.Close()
					for _, b := range bytes.Split(buf, []byte{'\n'}) {
						if len(b) > 0 {
							seen++
						}
					}
				}
				last = info.Size()
			}
			time.Sleep(10 * time.Millisecond)
		}
		if seen < n {
			readErr = fmt.Errorf("seen %d entries, expected %d", seen, n)
		}
	}()

	wg.Wait()
	ensureAbsent(fpath)
	if appendErr != nil {
		return appendErr
	}
	if readErr != nil {
		return readErr
	}
	return nil
}
