package autochecks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	autotests "github.com/mimic/test/automated/declarations"
)

func CheckLargeWriteTest(base string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  "CheckLargeWriteTest",
		State: autotests.TestFailed,
	}

	baseFName := path.Join(base, "largefile.largewrite")
	opts := &autotests.FilePreparationOptions{}
	prep := autotests.PrepareFileForTest(baseFName, opts.RequestedBy("Large Write"))
	if prep.State == autotests.TestFailed {
		res.State = autotests.TestChildFailed
		res.Error = fmt.Errorf("file preparation failed: %w", prep.Error)
		return res
	}
	res.Children = append(res.Children, prep)

	defer func() {
		_ = os.RemoveAll(baseFName)
	}()

	largeWriteRes := autotests.Timer(baseFName, LargeWriteTest)
	if largeWriteRes.State == autotests.TestFailed {

		res.Error = fmt.Errorf("large write test failed: %w", largeWriteRes.Error)
		return res
	}
	res.Children = append(res.Children, largeWriteRes)

	largeSizeRes := autotests.Timer(baseFName, LargeFileSizeTest)
	if largeSizeRes.State == autotests.TestFailed {
		res.Error = fmt.Errorf("large file size test failed: %w", largeSizeRes.Error)
		return res
	}
	res.Children = append(res.Children, largeSizeRes)

	// chunkSizes := []int{4096, 1024 * 1024}
	chunkSizes := []int{4 * 1024, 64 * 1024, 1024 * 1024}
	for _, sz := range chunkSizes {
		chunkedReadRes := autotests.Timer(baseFName, func(p string) *autotests.AutoTestResult {
			return ChunkedReadTest(p, sz)
		})
		res.Children = append(res.Children, chunkedReadRes)
		if chunkedReadRes.State == autotests.TestFailed {
			res.Error = fmt.Errorf("chunked read test (size %d) failed: %w", sz, chunkedReadRes.Error)
			return res
		}
	}

	res.State = autotests.TestPassed
	return res
}

func LargeWriteTest(base string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  "large write",
		State: autotests.TestFailed,
	}
	data := bytes.Repeat([]byte{0}, 1024*1024) // 1 MiB

	f, err := os.OpenFile(base, os.O_WRONLY, 0644)
	if err != nil {
		res.Error = fmt.Errorf("failed to open base dir: %w", err)
		return res
	}
	defer f.Close()

	for i := range 10 {
		_, err = f.Write(data)
		if err != nil {
			res.Error = fmt.Errorf("failed to write data chunk %d: %w", i, err)
			return res
		}
	}

	res.State = autotests.TestPassed
	return res
}

func LargeFileSizeTest(base string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  "large file size check",
		State: autotests.TestFailed,
	}

	retries := 5

sizeCheck:
	info, err := os.Stat(base)
	if err != nil {
		res.Error = fmt.Errorf("failed to stat file: %w", err)
		return res
	}
	if info.Size() != 10*1024*1024 {
		if retries != 0 {
			retries--
			time.Sleep(500 * time.Millisecond)
			goto sizeCheck
		}

		res.Error = fmt.Errorf("incorrect file size: got %d, want %d", info.Size(), 10*1024*1024)
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func ChunkedReadTest(base string, chunkSize int) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  fmt.Sprintf("chunked read %d", chunkSize),
		State: autotests.TestFailed,
	}

	f, err := os.Open(base)
	if err != nil {
		res.Error = fmt.Errorf("failed to open file: %w", err)
		return res
	}
	defer f.Close()

	buf := make([]byte, chunkSize)
	var total int64
	for {
		n, err := f.Read(buf)
		if n > 0 {
			for i := range n {
				if buf[i] != 0 {
					res.Error = fmt.Errorf("non-zero byte at offset %d (chunk %d)", total+int64(i), chunkSize)
					return res
				}
				total += int64(n)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			res.Error = fmt.Errorf("read error for chunk size %d: %w", chunkSize, err)
			return res
		}
	}
	res.State = autotests.TestPassed
	return res
}

// CheckLargeWrite writes a 10 MiB file to validate large writes.
func CheckLargeWrite(base string) (retErr error) {
	fpath := path.Join(base, "largefile.largewrite")
	var info os.FileInfo
	var err error
	var retries int = 5
	var out *os.File
	chunkSizes := []int{1, 4096, 1024 * 1024}
	expected := int64(10 * 1024 * 1024)
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
		fmt.Printf("Incorrect size %d, trying againn", info.Size())
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

	fmt.Printf("Succeeded after %d retriesn", 5-retries+1)

	// read-back checks with different chunk sizes (small and large)

	for _, sz := range chunkSizes {
		s = time.Now()
		f, err := os.Open(fpath)
		if err != nil {
			retErr = fmt.Errorf("failed to open file for read-%d: %w", sz, err)
			summarizeSub("Large write", fmt.Sprintf("read-%d", sz), retErr, time.Since(s))
			goto cleanup
		}
		buf := make([]byte, sz)
		var total int64
		for {
			n, err := f.Read(buf)
			if n > 0 {
				for i := 0; i < n; i++ {
					if buf[i] != 0 {
						retErr = fmt.Errorf("non-zero byte at offset %d (chunk %d)", total+int64(i), sz)
						_ = f.Close()
						summarizeSub("Large write", fmt.Sprintf("read-%d", sz), retErr, time.Since(s))
						goto cleanup
					}
				}
				total += int64(n)
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				retErr = fmt.Errorf("read error for chunk size %d: %w", sz, err)
				_ = f.Close()
				summarizeSub("Large write", fmt.Sprintf("read-%d", sz), retErr, time.Since(s))
				goto cleanup
			}
		}
		_ = f.Close()
		if total != expected {
			retErr = fmt.Errorf("read %d bytes, expected %d for chunk size %d", total, expected, sz)
			summarizeSub("Large write", fmt.Sprintf("read-%d", sz), retErr, time.Since(s))
			goto cleanup
		}
		summarizeSub("Large write", fmt.Sprintf("read-%d", sz), nil, time.Since(s))
	}

cleanup:
	if out != nil {
		_ = out.Close()
	}
	_ = os.RemoveAll(fpath)
	return
}
