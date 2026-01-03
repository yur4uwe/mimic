package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	autochecks "github.com/mimic/test/automated/checks"
	autotests "github.com/mimic/test/automated/declarations"
)

type checkFunc func(string) error

func runTest(fn func(string) *autotests.AutoTestResult, mount string) {
	res := autotests.Timer(mount, fn)
	UnwrapResult(res, 0)
}

func UnwrapResult(res *autotests.AutoTestResult, depth int) {
	if res.Passed {
		fmt.Printf("%s[PASS] %s (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Duration)
	} else {
		fmt.Printf("%s[FAIL] %s: Error encountered: %v (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Error, res.Duration)
	}
	if len(res.Children) == 0 {
		return
	}
	for _, child := range res.Children {
		UnwrapResult(child, depth+1)
	}
}

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

// TODO:
// - add out of bound read range requests test
// - test communication between host and guest filesystem
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

	// runCheck("File operations", autochecks.CheckFileOps, mount)
	runTest(autochecks.CheckFileOpsTest, mount)
	// runCheck("Large write", autochecks.CheckLargeWrite, mount)
	runTest(autochecks.CheckLargeWriteTest, mount)
	// runCheck("Open flags", autochecks.CheckOpenFlags, mount)
	runTest(autochecks.CheckOpenFlagsTest, mount)
	// runCheck("Truncate", autochecks.CheckTruncate, mount)
	runTest(autochecks.CheckTruncateTest, mount)
	// runCheck("Concurrent append/read", autochecks.CheckConcurrentAppendRead, mount)
	runTest(autochecks.CheckConcurrentTest, mount)

	fmt.Println("Unmounting...")
	time.Sleep(50 * time.Millisecond)
}
