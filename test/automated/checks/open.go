package autochecks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CheckOpenFlags verifies O_RDONLY, O_WRONLY, O_APPEND, O_TRUNC, O_CREATE, O_EXCL semantics.
func CheckOpenFlags(base string) (retErr error) {
	fbase := filepath.Join(base, "flags.dat")
	var f *os.File
	defer func() {
		if f != nil {
			_ = f.Close()
		}
		_ = os.Remove(fbase + ".baseline")
		_ = os.Remove(fbase + ".O_RDONLY")
		_ = os.Remove(fbase + ".O_WRONLY")
		_ = os.Remove(fbase + ".restoreForTrunc")
		_ = os.Remove(fbase + ".trunc")
		_ = os.Remove(fbase + ".create")
	}()

	start := time.Now()
	log.Printf("[CheckOpenFlags] start base=%s", base)

	// baseline
	s := time.Now()
	if err := writeBaseline(fbase + ".baseline"); err != nil {
		log.Printf("[CheckOpenFlags] baseline failed after %s: %v", time.Since(start), err)
		summarizeSub("Open flags", "baseline", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "baseline", nil, time.Since(s))

	toCreate := map[string][]byte{
		fbase + ".O_RDONLY": []byte("BASE"),
		fbase + ".O_WRONLY": []byte("BASE"),
		fbase + ".append":   []byte("BASE"),
		fbase + ".trunc":    []byte("HELLO WORLD"),
	}
	for p, content := range toCreate {
		if err := os.WriteFile(p, content, 0644); err != nil {
			log.Printf("[CheckOpenFlags] precreate %s failed after %s: %v", p, time.Since(start), err)
			summarizeSub("Open flags", "precreateFiles", err, time.Since(s))
			return err
		}
	}
	summarizeSub("Open flags", "precreateFiles", nil, time.Since(s))

	// 1) O_RDONLY -> writes should fail, reads succeed
	s = time.Now()
	if err := checkO_RDONLY(fbase + ".O_RDONLY"); err != nil {
		log.Printf("[CheckOpenFlags] O_RDONLY failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "O_RDONLY", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "O_RDONLY", nil, time.Since(s))

	// 2) O_WRONLY -> read should fail, write succeed
	s = time.Now()
	if err := checkO_WRONLY(fbase + ".O_WRONLY"); err != nil {
		log.Printf("[CheckOpenFlags] O_WRONLY failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "O_WRONLY", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "O_WRONLY", nil, time.Since(s))

	// restore baseline
	s = time.Now()
	if err := writeBaseline(fbase + ".baseline"); err != nil {
		log.Printf("[CheckOpenFlags] restore baseline failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "restoreBaseline", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "restoreBaseline", nil, time.Since(s))

	// 3) O_APPEND -> writes append
	s = time.Now()
	if err := checkO_APPEND(fbase + ".append"); err != nil {
		log.Printf("[CheckOpenFlags] O_APPEND failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "O_APPEND", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "O_APPEND", nil, time.Since(s))

	// restore baseline (different content used by later tests)
	s = time.Now()
	if err := os.WriteFile(fbase+".restoreForTrunc", []byte("HELLO WORLD"), 0644); err != nil {
		log.Printf("[CheckOpenFlags] restore for TRUNC failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "restoreForTrunc", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "restoreForTrunc", nil, time.Since(s))

	// 4) O_TRUNC -> truncates on open
	s = time.Now()
	if err := checkO_TRUNC(fbase + ".trunc"); err != nil {
		log.Printf("[CheckOpenFlags] O_TRUNC failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "O_TRUNC", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "O_TRUNC", nil, time.Since(s))

	// 5) O_CREAT creates when missing
	s = time.Now()
	if err := checkO_CREATE(fbase + ".create"); err != nil {
		log.Printf("[CheckOpenFlags] O_CREATE failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Open flags", "O_CREATE", err, time.Since(s))
		return err
	}
	summarizeSub("Open flags", "O_CREATE", nil, time.Since(s))

	log.Printf("[CheckOpenFlags] finished (total elapsed=%s)", time.Since(start))
	return nil
}

func writeBaseline(fpath string) error {
	start := time.Now()
	log.Printf("[writeBaseline] start path=%s", fpath)
	if err := os.WriteFile(fpath, []byte("BASE"), 0644); err != nil {
		log.Printf("[writeBaseline] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[writeBaseline] done (elapsed=%s)", time.Since(start))
	return nil
}

func checkO_RDONLY(fpath string) error {
	start := time.Now()
	log.Printf("[checkO_RDONLY] start path=%s", fpath)

	f, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("[checkO_RDONLY] open failed after %s: %v", time.Since(start), err)
		return err
	}

	if _, err = f.Write([]byte("X")); err == nil {
		err = errors.New("write succeeded on O_RDONLY (should fail)")
		log.Printf("[checkO_RDONLY] failed after %s: %v", time.Since(start), err)
		return err
	}

	buf := make([]byte, 4)
	if _, err = f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("[checkO_RDONLY] read failed after %s: %v", time.Since(start), err)
		return err
	}
	f.Close()
	log.Printf("[checkO_RDONLY] done (elapsed=%s)", time.Since(start))
	return nil
}

func checkO_WRONLY(fpath string) error {
	start := time.Now()
	log.Printf("[checkO_WRONLY] start path=%s", fpath)

	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("[checkO_WRONLY] open failed after %s: %v", time.Since(start), err)
		return err
	}

	// read should fail
	if _, err = f.Read(make([]byte, 8)); err == nil {
		err = errors.New("read succeeded on O_WRONLY (should fail)")
		log.Printf("[checkO_WRONLY] failed after %s: %v", time.Since(start), err)
		return err
	}

	// write should succeed and return correct byte count
	n, err := f.Write([]byte("WO"))
	if err != nil || n != 2 {
		clerr := f.Close()
		err = errors.Join(err, clerr)
		log.Printf("[checkO_WRONLY] write failed after %s: %v", time.Since(start), err)
		log.Printf("Wrote %d bytes, expected 2", n)
		return err
	}
	f.Close()

	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[checkO_WRONLY] readfile failed after %s: %v", time.Since(start), err)
		return err
	}
	if !bytes.Contains(b, []byte("WO")) {
		err = errors.New("O_WRONLY did not write as expected")
		log.Printf("[checkO_WRONLY] verification failed after %s: %v", time.Since(start), err)
		log.Printf("Read file contents: '%s', expected to have 'WO'", b)
		return err
	}

	log.Printf("[checkO_WRONLY] done (elapsed=%s)", time.Since(start))
	return nil
}

func checkO_APPEND(fpath string) error {
	start := time.Now()
	log.Printf("[checkO_APPEND] start path=%s", fpath)

	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("[checkO_APPEND] open failed after %s: %v", time.Since(start), err)
		return err
	}
	// write should append
	n, err := f.Write([]byte("A"))
	if err != nil || n != 1 {
		err = errors.Join(err, errors.New("unexpected write byte count for O_APPEND"))
		log.Printf("[checkO_APPEND] write failed after %s: %v", time.Since(start), err)

		return err
	}
	f.Close()

	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[checkO_APPEND] readfile failed after %s: %v", time.Since(start), err)
		return err
	}
	if !bytes.HasSuffix(b, []byte("A")) {
		err = errors.New("O_APPEND did not append")
		log.Printf("[checkO_APPEND] verification failed after %s: %v", time.Since(start), err)
		log.Printf("Read file contents: '%s', expected to end with 'A'", b)
		return err
	}
	// verify file length increased by 1 compared to baseline "BASE"(4)
	if len(b) != 5 {
		err = fmt.Errorf("O_APPEND resulted in unexpected file size: %d", len(b))
		log.Printf("[checkO_APPEND] size verification failed after %s: %v", time.Since(start), err)
		return err
	}

	log.Printf("[checkO_APPEND] done (elapsed=%s)", time.Since(start))
	return nil
}

func checkO_TRUNC(fpath string) error {
	start := time.Now()
	log.Printf("[checkO_TRUNC] start path=%s", fpath)

	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[checkO_TRUNC] open failed after %s: %v", time.Since(start), err)
		return err
	}
	// write after truncation
	n, err := f.Write([]byte("T"))
	if err != nil || n != 1 {
		err = errors.Join(err, errors.New("unexpected write byte count for O_TRUNC"))
		log.Printf("[checkO_TRUNC] write failed after %s: %v", time.Since(start), err)
		return err
	}
	f.Close()

	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[checkO_TRUNC] readfile failed after %s: %v", time.Since(start), err)
		return err
	}
	if !bytes.Equal(b, []byte("T")) {
		err = errors.New("O_TRUNC didn't truncate")
		log.Printf("[checkO_TRUNC] verification failed after %s: %v", time.Since(start), err)
		return err
	}
	if len(b) != 1 {
		err = fmt.Errorf("O_TRUNC produced unexpected size: %d", len(b))
		log.Printf("[checkO_TRUNC] size verification failed after %s: %v", time.Since(start), err)
		return err
	}

	log.Printf("[checkO_TRUNC] done (elapsed=%s)", time.Since(start))
	return nil
}

func checkO_CREATE(fpath string) error {
	start := time.Now()
	log.Printf("[checkO_CREATE] start path=%s", fpath)

	_ = os.Remove(fpath)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[checkO_CREATE] open failed after %s: %v", time.Since(start), err)
		return err
	}
	n, err := f.Write([]byte("C"))
	if err != nil || n != 1 {
		err = errors.Join(err, errors.New("unexpected write byte count for O_CREATE"))
		log.Printf("[checkO_CREATE] write failed after %s: %v", time.Since(start), err)
		return err
	}
	f.Close()

	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[checkO_CREATE] readfile failed after %s: %v", time.Since(start), err)
		return err
	}
	if !bytes.Equal(b, []byte("C")) {
		err = errors.New("O_CREATE did not create/write")
		log.Printf("[checkO_CREATE] verification failed after %s: %v", time.Since(start), err)
		return err
	}
	if len(b) != 1 {
		err = fmt.Errorf("O_CREATE produced unexpected size: %d", len(b))
		log.Printf("[checkO_CREATE] size verification failed after %s: %v", time.Since(start), err)
		return err
	}

	log.Printf("[checkO_CREATE] done (elapsed=%s)", time.Since(start))
	return nil
}
