package main

import (
	"fmt"
	"os"
	"time"

	autochecks "github.com/mimic/test/automated/checks"
)

type checkFunc func(string) error

func runCheck(name string, fn checkFunc, mount string) {
	start := time.Now()

	err := fn(mount)

	dur := time.Since(start)
	if err != nil {
		fmt.Printf("[FAIL] %s: Error encountered: %v (took %s)\n", name, err, dur)
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
		fmt.Printf("mountpoint not accessible or not a dir: %v\n", err)
		return
	}

	fmt.Println("Mountpoint is accessible.")

	runCheck("File operations", autochecks.CheckFileOps, mount)
	runCheck("Large write", autochecks.CheckLargeWrite, mount)
	runCheck("Open flags", autochecks.CheckOpenFlags, mount)
	runCheck("Truncate", autochecks.CheckTruncate, mount)
	runCheck("Concurrent append/read", autochecks.CheckConcurrentAppendRead, mount)

	fmt.Println("Unmounting...")
	time.Sleep(50 * time.Millisecond)
}
