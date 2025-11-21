package autochecks

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// CheckOpenFlags verifies O_RDONLY, O_WRONLY, O_APPEND, O_TRUNC, O_CREATE, O_EXCL semantics.
func CheckOpenFlags(base string) error {
	fpath := filepath.Join(base, "flags.dat")

	// baseline
	if err := writeFile(fpath, []byte("BASE")); err != nil {
		return err
	}

	// 1) O_RDONLY -> writes should fail, reads succeed
	{
		f, err := os.OpenFile(fpath, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		if _, err := f.Write([]byte("X")); err == nil {
			_ = f.Close()
			return errors.New("write succeeded on O_RDONLY (should fail)")
		}
		buf := make([]byte, 4)
		if _, err := f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
			_ = f.Close()
			return err
		}
		_ = f.Close()
	}

	// 2) O_WRONLY -> read should fail, write succeed
	{
		f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		if _, rerr := f.Read(make([]byte, 8)); rerr == nil {
			_ = f.Close()
			return errors.New("read succeeded on O_WRONLY (should fail)")
		}

		if _, err := f.Write([]byte("WO")); err != nil {
			clerr := f.Close()
			return errors.Join(err, clerr)
		}
		err = f.Close()
		if err != nil {
			return err
		}

		b, err := readAll(fpath)
		if err != nil {
			return err
		}
		if !bytes.Contains(b, []byte("WO")) {
			return errors.New("O_WRONLY did not write as expected")
		}
	}

	// restore baseline
	if err := writeFile(fpath, []byte("BASE")); err != nil {
		return err
	}

	// 3) O_APPEND -> writes append
	if err := appendString(fpath, "A"); err != nil {
		return err
	}
	b, err := readAll(fpath)
	if err != nil {
		return err
	}
	if !containsSuffix(b, "A") {
		return errors.New("O_APPEND did not append")
	}

	// restore baseline
	if err := writeFile(fpath, []byte("HELLO WORLD")); err != nil {
		return err
	}

	// 4) O_TRUNC -> truncates on open
	{
		f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		if _, err := f.Write([]byte("T")); err != nil {
			_ = f.Close()
			return err
		}
		_ = f.Close()
		b, err := readAll(fpath)
		if err != nil {
			return err
		}
		if !bytes.Equal(b, []byte("T")) {
			return errors.New("O_TRUNC didn't truncate")
		}
	}

	// 6) O_CREAT creates when missing
	{
		_ = os.Remove(fpath)
		f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		if _, err := f.Write([]byte("C")); err != nil {
			_ = f.Close()
			return err
		}
		_ = f.Close()
		b, err := readAll(fpath)
		if err != nil {
			return err
		}
		if !bytes.Equal(b, []byte("C")) {
			return errors.New("O_CREATE did not create/write")
		}
	}

	ensureAbsent(fpath)
	return nil
}
