package autochecks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CheckTruncate verifies truncating down and up works.
func CheckTruncate(base string) (retErr error) {
	fpath := filepath.Join(base, "tfile")
	start := time.Now()
	log.Printf("[CheckTruncate] start base=%s", base)
	defer func() {
		_ = os.RemoveAll(fpath)
		log.Printf("[CheckTruncate] finished (total elapsed=%s) err=%v", time.Since(start), retErr)
	}()

	// create
	s := time.Now()
	if err := createFileForTruncate(fpath + ".down"); err != nil {
		log.Printf("[CheckTruncate] create down failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Truncate", "createFile", err, time.Since(s))
		return err
	}
	summarizeSub("Truncate", "createFile", nil, time.Since(s))

	s = time.Now()
	if err := createFileForTruncate(fpath + ".up"); err != nil {
		log.Printf("[CheckTruncate] create up failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Truncate", "createFile", err, time.Since(s))
		return err
	}
	summarizeSub("Truncate", "createFile", nil, time.Since(s))

	// truncate down
	s = time.Now()
	if err := truncateDown(fpath + ".down"); err != nil {
		log.Printf("[CheckTruncate] truncateDown failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Truncate", "truncateDown", err, time.Since(s))
		return err
	}
	summarizeSub("Truncate", "truncateDown", nil, time.Since(s))

	// truncate up
	s = time.Now()
	if err := truncateUp(fpath + ".up"); err != nil {
		log.Printf("[CheckTruncate] truncateUp failed (elapsed=%s): %v", time.Since(start), err)
		summarizeSub("Truncate", "truncateUp", err, time.Since(s))
		return err
	}
	summarizeSub("Truncate", "truncateUp", nil, time.Since(s))

	return nil
}

func createFileForTruncate(fpath string) error {
	start := time.Now()
	log.Printf("[createFileForTruncate] start path=%s", fpath)
	if err := os.WriteFile(fpath, []byte("0123456789"), 0644); err != nil {
		log.Printf("[createFileForTruncate] failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[createFileForTruncate] done (elapsed=%s)", time.Since(start))
	return nil
}

func truncateDown(fpath string) error {
	start := time.Now()
	log.Printf("[truncateDown] start path=%s", fpath)
	if err := os.Truncate(fpath, 4); err != nil {
		log.Printf("[truncateDown] failed after %s: %v", time.Since(start), err)
		return err
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[truncateDown] read failed after %s: %v", time.Since(start), err)
		return err
	}
	if len(b) != 4 {
		err = fmt.Errorf("truncate down wrong size: %d", len(b))
		log.Printf("[truncateDown] verification failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[truncateDown] done (elapsed=%s)", time.Since(start))
	return nil
}

func truncateUp(fpath string) error {
	start := time.Now()
	log.Printf("[truncateUp] start path=%s", fpath)
	if err := os.Truncate(fpath, 10); err != nil {
		log.Printf("[truncateUp] failed after %s: %v", time.Since(start), err)
		return err
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		log.Printf("[truncateUp] read failed after %s: %v", time.Since(start), err)
		return err
	}
	if len(b) != 10 {
		err = fmt.Errorf("truncate up wrong size: %d", len(b))
		log.Printf("[truncateUp] verification failed after %s: %v", time.Since(start), err)
		return err
	}
	log.Printf("[truncateUp] done (elapsed=%s)", time.Since(start))
	return nil
}
