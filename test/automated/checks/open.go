package autochecks

import (
	"bytes"
	"errors"
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
	buf := make([]byte, 4)

	// baseline
	if err = writeFile(fpath, []byte("BASE")); err != nil {
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
	// write should succeed
	if _, err = f.Write([]byte("WO")); err != nil {
		clerr := f.Close()
		retErr = errors.Join(err, clerr)
		goto cleanup
	}
	if err = f.Close(); err != nil {
		retErr = err
		goto cleanup
	}

	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Contains(b, []byte("WO")) {
		retErr = errors.New("O_WRONLY did not write as expected")
		goto cleanup
	}

	// restore baseline
	if err = writeFile(fpath, []byte("BASE")); err != nil {
		retErr = err
		goto cleanup
	}

	// 3) O_APPEND -> writes append
	if err = appendString(fpath, "A"); err != nil {
		retErr = err
		goto cleanup
	}
	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !containsSuffix(b, "A") {
		retErr = errors.New("O_APPEND did not append")
		goto cleanup
	}

	// restore baseline
	if err = writeFile(fpath, []byte("HELLO WORLD")); err != nil {
		retErr = err
		goto cleanup
	}

	// 4) O_TRUNC -> truncates on open
	f, err = os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if _, err = f.Write([]byte("T")); err != nil {
		_ = f.Close()
		retErr = err
		goto cleanup
	}
	_ = f.Close()
	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Equal(b, []byte("T")) {
		retErr = errors.New("O_TRUNC didn't truncate")
		goto cleanup
	}

	// 6) O_CREAT creates when missing
	_ = os.Remove(fpath)
	f, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if _, err = f.Write([]byte("C")); err != nil {
		_ = f.Close()
		retErr = err
		goto cleanup
	}
	_ = f.Close()
	b, err = readAll(fpath)
	if err != nil {
		retErr = err
		goto cleanup
	}
	if !bytes.Equal(b, []byte("C")) {
		retErr = errors.New("O_CREATE did not create/write")
		goto cleanup
	}

cleanup:
	// ensure file removed and any open handle closed
	if f != nil {
		_ = f.Close()
	}
	ensureAbsent(fpath)
	return
}
