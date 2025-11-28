package main

import (
	"fmt"
	"os"
	"time"

	autochecks "github.com/mimic/test/automated/checks"
)

func runCheck(name string, fn func() error) {
	start := time.Now()
	err := fn()
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("[FAIL] %s: Error encoutered: %v (took %s)\n", name, err, dur)
	} else {
		fmt.Printf("[PASS] %s: Check succeeded (took %s)\n", name, dur)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s MOUNTPOINT\n", os.Args[0])
		os.Exit(2)
	}
	mount := os.Args[1]

	if fi, err := os.Stat(mount); err != nil || !fi.IsDir() {
		runCheck("Mountpoint accessibility", func() error {
			return fmt.Errorf("mountpoint not accessible or not a dir: %v", err)
		})
		return
	}

	fmt.Println("Mountpoint is accessible.")

	runCheck("File operations", func() error { return autochecks.CheckFileOps(mount) })
	runCheck("Large write", func() error { return autochecks.CheckLargeWrite(mount) })
	runCheck("Open flags", func() error { return autochecks.CheckOpenFlags(mount) })
	runCheck("Truncate", func() error { return autochecks.CheckTruncate(mount) })
	runCheck("Concurrent append/read", func() error { return autochecks.CheckConcurrentAppendRead(mount) })

	fmt.Println("Unmounting...")
}
