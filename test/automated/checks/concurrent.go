package autochecks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CheckConcurrentAppendRead performs concurrent appends and a reader tailing the file.
func CheckConcurrentAppendRead(base string) error {
	fpath := filepath.Join(base, "stream.txt")
	ensureAbsent(fpath)
	if err := writeFile(fpath, []byte{}); err != nil {
		return err
	}

	const n = 5
	var wg sync.WaitGroup
	wg.Add(2)

	appendCh := make(chan error, 1)
	readCh := make(chan error, 1)

	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				appendCh <- err
				return
			}
			line := fmt.Sprintf("x%02d\n", i)
			if _, err := f.Write([]byte(line)); err != nil {
				_ = f.Close()
				appendCh <- err
				return
			}
			if err := f.Close(); err != nil {
				appendCh <- err
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		appendCh <- nil
	}()

	go func() {
		defer wg.Done()
		seen := 0
		var last int64 = 0
		for seen < n {
			info, err := os.Stat(fpath)
			if err != nil {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			if info.Size() > last {
				f, err := os.Open(fpath)
				if err != nil {
					readCh <- err
					return
				}
				if _, err := f.Seek(last, io.SeekStart); err != nil {
					_ = f.Close()
					readCh <- err
					return
				}
				buf := make([]byte, info.Size()-last)
				if _, err := io.ReadFull(f, buf); err != nil && err != io.EOF {
					_ = f.Close()
					readCh <- err
					return
				}
				if err := f.Close(); err != nil {
					readCh <- err
					return
				}
				for _, b := range bytes.Split(buf, []byte{'\n'}) {
					if len(b) > 0 {
						seen++
					}
				}
				last = info.Size()
			}
			time.Sleep(20 * time.Millisecond)
		}
		if seen < n {
			readCh <- fmt.Errorf("seen %d entries, expected %d", seen, n)
			return
		}
		readCh <- nil
	}()

	wg.Wait()
	appendErr := <-appendCh
	readErr := <-readCh
	ensureAbsent(fpath)
	if appendErr != nil {
		return appendErr
	}
	if readErr != nil {
		return readErr
	}
	return nil
}
