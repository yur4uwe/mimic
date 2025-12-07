package autochecks

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

// CheckFileOps covers create, append, mkdir, move, rename and remove.
func CheckFileOps(base string) (retErr error) {
	f := path.Join(base, "basic.create") // initial file name indicates "create"
	dir := path.Join(base, "test_dir")
	renamed := path.Join(base, "basic.renamed")

	start := time.Now()
	log.Printf("[CheckFileOps] start base=%s", base)
	defer func() {
		_ = os.RemoveAll(f)
		_ = os.RemoveAll(renamed)
		_ = os.RemoveAll(dir)
		log.Printf("[CheckFileOps] finished (total elapsed=%s) err=%v", time.Since(start), retErr)
	}()

	// createAndWrite
	s := time.Now()
	if err := createAndWrite(f); err != nil {
		log.Printf("[CheckFileOps] createAndWrite failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "createAndWrite", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "createAndWrite", nil, time.Since(s))

	// appendLine
	s = time.Now()
	if err := appendLine(f); err != nil {
		log.Printf("[CheckFileOps] appendLine failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "appendLine", err, time.Since(s))
		return err
	}
	// rename file to indicate append was performed
	newName := f + ".append"
	if err := os.Rename(f, newName); err != nil {
		log.Printf("[CheckFileOps] rename after append failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "appendLine", err, time.Since(s))
		return err
	}
	f = newName
	summarizeSub("File operations", "appendLine", nil, time.Since(s))

	// statCheck
	s = time.Now()
	if err := statCheck(f); err != nil {
		log.Printf("[CheckFileOps] statCheck failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "statCheck", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "statCheck", nil, time.Since(s))

	// makeDir
	s = time.Now()
	if err := makeDir(dir); err != nil {
		log.Printf("[CheckFileOps] makeDir failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "makeDir", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "makeDir", nil, time.Since(s))

	// moveIntoDir
	s = time.Now()
	if err := moveIntoDir(f, dir); err != nil {
		log.Printf("[CheckFileOps] moveIntoDir failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "moveIntoDir", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "moveIntoDir", nil, time.Since(s))

	// renameOut
	s = time.Now()
	if err := renameOut(dir, filepath.Base(f), renamed); err != nil {
		log.Printf("[CheckFileOps] renameOut failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "renameOut", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "renameOut", nil, time.Since(s))

	// cleanup remove
	s = time.Now()
	if err := os.Remove(renamed); err != nil {
		log.Printf("[CheckFileOps] cleanup remove failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("File operations", "cleanupRemove", err, time.Since(s))
		return err
	}
	summarizeSub("File operations", "cleanupRemove", nil, time.Since(s))

	return nil
}

func createAndWrite(f string) error {
	start := time.Now()
	log.Printf("[createAndWrite] start path=%s", f)
	if err := os.WriteFile(f, []byte("hello world\n"), 0644); err != nil {
		log.Printf("[createAndWrite] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[createAndWrite] done (elapsed=%s)", time.Since(start))
	return nil
}

func appendLine(f string) error {
	start := time.Now()
	log.Printf("[appendLine] start path=%s", f)
	fh, err := os.OpenFile(f, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		log.Printf("[appendLine] open failed after %s: %v", time.Since(start), err)
		return err
	}
	defer fh.Close()
	if _, err := fh.Write([]byte("append-line\n")); err != nil {
		log.Printf("[appendLine] write failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[appendLine] done (elapsed=%s)", time.Since(start))
	return nil
}

func statCheck(f string) error {
	start := time.Now()
	log.Printf("[statCheck] start path=%s", f)
	if _, err := os.Stat(f); err != nil {
		log.Printf("[statCheck] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[statCheck] done (elapsed=%s)", time.Since(start))
	return nil
}

func makeDir(dir string) error {
	start := time.Now()
	log.Printf("[makeDir] start dir=%s", dir)
	if err := os.Mkdir(dir, 0755); err != nil {
		log.Printf("[makeDir] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[makeDir] done (elapsed=%s)", time.Since(start))
	return nil
}

func moveIntoDir(f, dir string) error {
	start := time.Now()
	dest := filepath.Join(dir, filepath.Base(f))
	log.Printf("[moveIntoDir] start %s -> %s", f, dest)
	if err := os.Rename(f, dest); err != nil {
		log.Printf("[moveIntoDir] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[moveIntoDir] done (elapsed=%s)", time.Since(start))
	return nil
}

func renameOut(dir, base, renamed string) error {
	start := time.Now()
	src := filepath.Join(dir, base)
	log.Printf("[renameOut] start %s -> %s", src, renamed)
	if err := os.Rename(src, renamed); err != nil {
		log.Printf("[renameOut] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[renameOut] done (elapsed=%s)", time.Since(start))
	return nil
}
