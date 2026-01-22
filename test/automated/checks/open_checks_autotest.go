package autochecks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	autotests "github.com/mimic/test/automated/declarations"
)

// The following are independent implementations of the open-flag checks.
// They follow the semantics in their names and return an error on failure.

func checkO_RDONLY_new(fpath string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "checkO_RDONLY_new", State: autotests.TestFailed}

	f, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		res.Error = fmt.Errorf("open O_RDONLY: %w", err)
		return res
	}
	defer f.Close()

	// write should fail
	if _, err = f.Write([]byte("X")); err == nil {
		res.Error = errors.New("write succeeded on O_RDONLY (should fail)")
		return res
	}

	// read should succeed (allow EOF)
	buf := make([]byte, 4)
	if _, err := f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
		res.Error = fmt.Errorf("read failed on O_RDONLY: %w", err)
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func checkO_WRONLY_new(fpath string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "checkO_WRONLY_new", State: autotests.TestFailed}

	f, err := os.OpenFile(fpath, os.O_WRONLY, 0644)
	if err != nil {
		res.Error = fmt.Errorf("open O_WRONLY: %w", err)
		return res
	}
	defer f.Close()

	// read should fail
	if _, err := f.Read(make([]byte, 8)); err == nil {
		res.Error = errors.New("read succeeded on O_WRONLY (should fail)")
		return res
	}

	// write should succeed
	n, err := f.Write([]byte("WO"))
	if err != nil || n != 2 {
		res.Error = fmt.Errorf("write failed on O_WRONLY: %w", err)
		return res
	}

	// verify file contains the write
	b, err := os.ReadFile(fpath)
	if err != nil {
		res.Error = fmt.Errorf("readfile failed after O_WRONLY: %w", err)
		return res
	}
	if !bytes.Contains(b, []byte("WO")) {
		res.Error = fmt.Errorf("O_WRONLY verification failed: content=%q", b)
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func checkO_APPEND_new(fpath string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "checkO_APPEND_new", State: autotests.TestFailed}

	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		res.Error = fmt.Errorf("open O_APPEND: %w", err)
		return res
	}
	defer f.Close()

	var buf []byte = make([]byte, 4)
	n, _ := f.Read(buf)
	fmt.Printf("bytes read: %d buf: %v\n", n, buf)

	// write should append
	n, err = f.Write([]byte("A"))
	if err != nil || n != 1 {
		res.Error = fmt.Errorf("write failed on O_APPEND: %w", err)
		return res
	}

	time.Sleep(100 * time.Millisecond)

	b, err := os.ReadFile(fpath)
	if err != nil {
		res.Error = fmt.Errorf("readfile failed after O_APPEND: %w", err)
		return res
	}
	if !bytes.Equal(b, []byte("BASEA")) {
		res.Error = fmt.Errorf("O_APPEND verification failed: content=%q", b)
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func checkO_TRUNC_new(fpath string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "checkO_TRUNC_new", State: autotests.TestFailed}

	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		res.Error = fmt.Errorf("open O_TRUNC: %w", err)
		return res
	}
	defer f.Close()

	// write after truncation
	n, err := f.Write([]byte("T"))
	if err != nil || n != 1 {
		res.Error = fmt.Errorf("write failed on O_TRUNC: %w", err)
		return res
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		res.Error = fmt.Errorf("readfile failed after O_TRUNC: %w", err)
		return res
	}
	if !bytes.Equal(b, []byte("T")) {
		res.Error = fmt.Errorf("O_TRUNC verification failed: content=%q", b)
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func checkO_CREATE_new(fpath string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "checkO_CREATE_new", State: autotests.TestFailed}

	_ = os.Remove(fpath)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		res.Error = fmt.Errorf("open O_CREATE: %w", err)
		return res
	}
	defer f.Close()

	n, err := f.Write([]byte("C"))
	if err != nil || n != 1 {
		res.Error = fmt.Errorf("write failed on O_CREATE: %w", err)
		return res
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		res.Error = fmt.Errorf("readfile failed after O_CREATE: %w", err)
		return res
	}
	if !bytes.Equal(b, []byte("C")) {
		res.Error = fmt.Errorf("O_CREATE verification failed: content=%q", b)
		return res
	}

	res.State = autotests.TestPassed
	return res
}
