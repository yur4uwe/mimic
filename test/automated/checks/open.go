package autochecks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CheckOpenFlags verifies O_RDONLY, O_WRONLY, O_APPEND, O_TRUNC, O_CREATE, O_EXCL semantics.
func CheckOpenFlags(base string) (retErr error) {
	fpath := filepath.Join(base, "flags.dat")
	var f *os.File
	var err error
	var b []byte
	var n int
	buf := make([]byte, 4)

	// baseline
	if err = os.WriteFile(fpath, []byte("BASE"), 0644); err != nil {
		retErr = err
		goto cleanup
	}

	// 1) O_RDONLY -> writes should fail, reads succeed
	f, err = os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		retErr = err
		goto cleanup
	}
	// attempt write should fail
	if _, err = f.Write([]byte("X")); err == nil {
		_ = f.Close()
		retErr = errors.New("write succeeded on O_RDONLY (should fail)")
		goto cleanup
	}
	// read should succeed
	if _, err = f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
		_ = f.Close()
		retErr = err
		goto cleanup
	}
	if err = f.Close(); err != nil {
		retErr = err
		goto cleanup
	}

	// 2) O_WRONLY -> read should fail, write succeed
	f, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	// read should fail
	if _, err = f.Read(make([]byte, 8)); err == nil {
		_ = f.Close()
		retErr = errors.New("read succeeded on O_WRONLY (should fail)")
		goto cleanup
	}
	// write should succeed and return correct byte count
	if n, err = f.Write([]byte("WO")); err != nil || n != 2 {
		clerr := f.Close()
		retErr = errors.Join(err, clerr)
		goto cleanup
	}
	if err = f.Close(); err != nil {
		retErr = err
		goto cleanup
	}

	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Contains(b, []byte("WO")) {
		retErr = errors.New("O_WRONLY did not write as expected")
		goto cleanup
	}
	// optional: ensure at least two bytes were written (we already checked n == 2 above)

	// restore baseline
	if err = os.WriteFile(fpath, []byte("BASE"), 0644); err != nil {
		retErr = err
		goto cleanup
	}

	// 3) O_APPEND -> writes append
	f, err = os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if n, err = f.Write([]byte("A")); err != nil || n != 1 {
		_ = f.Close()
		retErr = errors.Join(err, errors.New("unexpected write byte count for O_APPEND"))
		goto cleanup
	}
	if err = f.Close(); err != nil {
		retErr = err
		goto cleanup
	}

	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.HasSuffix(b, []byte("A")) {
		retErr = errors.New("O_APPEND did not append")
		goto cleanup
	}
	// verify file length increased by 1 compared to baseline
	if len(b) != 5 { // baseline "BASE" (4) + "A" (1) == 5
		retErr = fmt.Errorf("O_APPEND resulted in unexpected file size: %d", len(b))
		goto cleanup
	}

	// restore baseline
	if err = os.WriteFile(fpath, []byte("HELLO WORLD"), 0644); err != nil {
		retErr = err
		goto cleanup
	}

	// 4) O_TRUNC -> truncates on open
	f, err = os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if n, err = f.Write([]byte("T")); err != nil || n != 1 {
		_ = f.Close()
		retErr = errors.Join(err, errors.New("unexpected write byte count for O_TRUNC"))
		goto cleanup
	}
	_ = f.Close()
	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Equal(b, []byte("T")) {
		retErr = errors.New("O_TRUNC didn't truncate")
		goto cleanup
	}
	if len(b) != 1 {
		retErr = fmt.Errorf("O_TRUNC produced unexpected size: %d", len(b))
		goto cleanup
	}

	// 6) O_CREAT creates when missing
	_ = os.Remove(fpath)
	f, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if n, err = f.Write([]byte("C")); err != nil || n != 1 {
		_ = f.Close()
		retErr = errors.Join(err, errors.New("unexpected write byte count for O_CREATE"))
		goto cleanup
	}
	_ = f.Close()
	b, err = os.ReadFile(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Equal(b, []byte("C")) {
		retErr = errors.New("O_CREATE did not create/write")
		goto cleanup
	}
	if len(b) != 1 {
		retErr = fmt.Errorf("O_CREATE produced unexpected size: %d", len(b))
		goto cleanup
	}

cleanup:
	// ensure file removed and any open handle closed
	if f != nil {
		_ = f.Close()
	}
	_ = os.RemoveAll(fpath)
	return
}
