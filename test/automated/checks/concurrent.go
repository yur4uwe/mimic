package autochecks

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CheckConcurrentAppendRead performs concurrent appends and a reader tailing the file.
func CheckConcurrentAppendRead(base string) (retErr error) {
	fpath := filepath.Join(base, "stream.concurrent")
	_ = os.RemoveAll(fpath)

	const n = 5
	var wg sync.WaitGroup
	appendCh := make(chan error, 1)
	readCh := make(chan error, 1)

	start := time.Now()
	log.Printf("[CheckConcurrentAppendRead] start path=%s", fpath)

	if err := os.WriteFile(fpath, []byte{}, 0644); err != nil {
		log.Printf("[CheckConcurrentAppendRead] initial write failed (elapsed=%s): %v", time.Since(start), err)
		return err
	}

	wg.Add(2)
	appendStart := time.Now()
	go func() {
		defer wg.Done()
		log.Printf("[appendRoutine] start (n=%d)", n)
		for i := 0; i < n; i++ {
			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				appendCh <- fmt.Errorf("open for append error: %v", err)
				return
			}
			_, err = f.WriteString(fmt.Sprintf("x%02d\n", i))
			_ = f.Close()
			if err != nil {
				appendCh <- fmt.Errorf("write error: %v", err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		appendCh <- nil
		log.Printf("[appendRoutine] done (elapsed=%s)", time.Since(appendStart))
	}()

	readStart := time.Now()
	// reader: read newly appended bytes and update last by bytes actually read
	go func() {
		defer wg.Done()
		seen := 0
		var last int64 = 0
		startReader := time.Now()
		timeout := 5 * time.Second

		log.Printf("[readRoutine] start (expect=%d)", n)
		for seen < n {
			if time.Since(startReader) > timeout {
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
		log.Printf("[readRoutine] done (elapsed=%s)", time.Since(startReader))
	}()

	wg.Wait()
	appendErr := <-appendCh
	readErr := <-readCh

	summarizeSub("Concurrent append/read", "appendRoutine", appendErr, time.Since(appendStart))
	summarizeSub("Concurrent append/read", "readRoutine", readErr, time.Since(readStart))

	if appendErr != nil {
		retErr = appendErr
		log.Printf("[CheckConcurrentAppendRead] append error (elapsed=%s): %v", time.Since(start), appendErr)
		goto cleanup
	}
	if readErr != nil {
		retErr = readErr
		log.Printf("[CheckConcurrentAppendRead] read error (elapsed=%s): %v", time.Since(start), readErr)
		goto cleanup
	}

cleanup:
	_ = os.RemoveAll(fpath)
	log.Printf("[CheckConcurrentAppendRead] finished (total elapsed=%s) err=%v", time.Since(start), retErr)
	return
}

func summarizeSub(parent, name string, err error, dur time.Duration) {
	if err != nil {
		log.Printf("[FAIL] %s/%s: Error encountered: %v (took %s)", parent, name, err, dur)
	} else {
		log.Printf("[PASS] %s/%s: Check succeeded (took %s)", parent, name, dur)
	}
}
