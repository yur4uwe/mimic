package autochecks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// The following are independent implementations of the open-flag checks.
// They follow the semantics in their names and return an error on failure.

func checkO_RDONLY_new(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open O_RDONLY: %w", err)
	}
	defer f.Close()

	// write should fail
	if _, err = f.Write([]byte("X")); err == nil {
		return errors.New("write succeeded on O_RDONLY (should fail)")
	}

	// read should succeed (allow EOF)
	buf := make([]byte, 4)
	if _, err := f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read failed on O_RDONLY: %w", err)
	}
	return nil
}

func checkO_WRONLY_new(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open O_WRONLY: %w", err)
	}
	defer f.Close()

	// read should fail
	if _, err := f.Read(make([]byte, 8)); err == nil {
		return errors.New("read succeeded on O_WRONLY (should fail)")
	}

	// write should succeed
	n, err := f.Write([]byte("WO"))
	if err != nil || n != 2 {
		return fmt.Errorf("write failed on O_WRONLY: %w", err)
	}

	// verify file contains the write
	b, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("readfile failed after O_WRONLY: %w", err)
	}
	if !bytes.Contains(b, []byte("WO")) {
		return fmt.Errorf("O_WRONLY verification failed: content=%q", b)
	}
	return nil
}

func checkO_APPEND_new(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open O_APPEND: %w", err)
	}
	defer f.Close()

	// write should append
	n, err := f.Write([]byte("A"))
	if err != nil || n != 1 {
		return fmt.Errorf("write failed on O_APPEND: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	b, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("readfile failed after O_APPEND: %w", err)
	}
	if !bytes.Equal(b, []byte("BASEA")) {
		return fmt.Errorf("O_APPEND verification failed: content=%q", b)
	}

	return nil
}

func checkO_TRUNC_new(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open O_TRUNC: %w", err)
	}
	defer f.Close()

	// write after truncation
	n, err := f.Write([]byte("T"))
	if err != nil || n != 1 {
		return fmt.Errorf("write failed on O_TRUNC: %w", err)
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("readfile failed after O_TRUNC: %w", err)
	}
	if !bytes.Equal(b, []byte("T")) {
		return fmt.Errorf("O_TRUNC verification failed: content=%q", b)
	}
	return nil
}

func checkO_CREATE_new(fpath string) error {
	// ensure missing
	_ = os.Remove(fpath)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open O_CREATE: %w", err)
	}
	defer f.Close()

	n, err := f.Write([]byte("C"))
	if err != nil || n != 1 {
		return fmt.Errorf("write failed on O_CREATE: %w", err)
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("readfile failed after O_CREATE: %w", err)
	}
	if !bytes.Equal(b, []byte("C")) {
		return fmt.Errorf("O_CREATE verification failed: content=%q", b)
	}
	return nil
}
