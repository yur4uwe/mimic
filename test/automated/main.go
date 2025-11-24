package main

import (
	"fmt"
	"os"

	autochecks "github.com/mimic/test/automated/checks"
)

func assert(name string, err error) {
	if err != nil {
		fmt.Printf("[FAIL] %s: Error encoutered: %v\n", name, err)
	} else {
		fmt.Printf("[PASS] %s: Check succeeded\n", name)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s MOUNTPOINT\n", os.Args[0])
		os.Exit(2)
	}
	mount := os.Args[1]

	if fi, err := os.Stat(mount); err != nil || !fi.IsDir() {
		assert("Mountpoint accessibility", fmt.Errorf("mountpoint not accessible or not a dir: %v", err))
		return
	}

	fmt.Println("Mountpoint is accessible.")

	assert("File operations", autochecks.CheckFileOps(mount))
	assert("Large write", autochecks.CheckLargeWrite(mount))
	assert("Open flags", autochecks.CheckOpenFlags(mount))
	assert("Truncate", autochecks.CheckTruncate(mount))
	assert("Concurrent append/read", autochecks.CheckConcurrentAppendRead(mount))

	fmt.Println("Unmounting...")
}
