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
func CheckConcurrentAppendRead(base string) (retErr error) {
	fpath := filepath.Join(base, "stream.txt")
	ensureAbsent(fpath)

	const n = 5
	var wg sync.WaitGroup
	appendCh := make(chan error, 1)
	readCh := make(chan error, 1)

	var appendErr, readErr error

	if err := writeFile(fpath, []byte{}); err != nil {
		retErr = err
		goto cleanup
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range n {
			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				appendCh <- err
				return
			}
			// write a line
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
			time.Sleep(50 * time.Millisecond)
		}
		appendCh <- nil
	}()

	// reader: read newly appended bytes and update last by bytes actually read
	go func() {
		defer wg.Done()
		seen := 0
		var last int64 = 0
		start := time.Now()
		timeout := 5 * time.Second

		for seen < n {
			// timeout to avoid infinite hang
			if time.Since(start) > timeout {
				readCh <- fmt.Errorf("timeout waiting for %d entries, seen %d", n, seen)
				return
			}

			info, err := os.Stat(fpath)
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			// check if file size has increased
			if info.Size() > last {
				f, err := os.Open(fpath)
				if err != nil {
					readCh <- err
					return
				}
				// seek to last read position
				if _, err := f.Seek(last, io.SeekStart); err != nil {
					_ = f.Close()
					readCh <- err
					return
				}
				toRead := int(info.Size() - last)
				if toRead <= 0 {
					_ = f.Close()
					time.Sleep(50 * time.Millisecond)
					continue
				}
				// read new data
				buf := make([]byte, toRead)
				nread, err := f.Read(buf)
				if err != nil && err != io.EOF {
					_ = f.Close()
					readCh <- err
					return
				}
				_ = f.Close()

				if nread > 0 {
					parts := bytes.Split(buf[:nread], []byte{'\n'})
					for _, p := range parts {
						if len(p) > 0 {
							seen++
						}
					}
					last += int64(nread)
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		readCh <- nil
	}()

	wg.Wait()
	appendErr = <-appendCh
	readErr = <-readCh
	if appendErr != nil {
		retErr = appendErr
		goto cleanup
	}
	if readErr != nil {
		retErr = readErr
		goto cleanup
	}

cleanup:
	ensureAbsent(fpath)
	return
}
