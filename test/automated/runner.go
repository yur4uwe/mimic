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

type TestResultAggregate struct {
	Passed   uint
	Skipped  uint
	Failed   uint
	Duration time.Duration
}

func UnwrapResult(res *autotests.AutoTestResult, aggr *TestResultAggregate, depth int) {
	if aggr == nil {
		panic("aggr cannot be nil")
	}

	switch res.State {
	case autotests.TestPassed:
		fmt.Printf("%s[PASS] %s (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Duration)
		aggr.Passed++
	case autotests.TestSkipped:
		fmt.Printf("%s[SKIP] %s (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Duration)
		aggr.Skipped++
	case autotests.TestFailed:
		fmt.Printf("%s[FAIL] %s: Error encountered: %v (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Error, res.Duration)
		aggr.Failed++
	case autotests.TestChildFailed:
		fmt.Printf("%s[FAIL] %s: One or more child tests failed (took %s)\n", strings.Repeat("  ", depth), res.Name, res.Duration)
		aggr.Skipped++
	default:
		panic("unknown test result state")
	}
	aggr.Duration += res.Duration
	if len(res.Children) == 0 {
		return
	}
	for _, child := range res.Children {
		UnwrapResult(child, aggr, depth+1)
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

	aggregate := &TestResultAggregate{}

	for _, callback := range []func(string) *autotests.AutoTestResult{
		autochecks.CheckFileOpsTest,
		autochecks.CheckLargeWriteTest,
		autochecks.CheckOpenFlagsTest,
		autochecks.CheckTruncateTest,
		autochecks.CheckConcurrentTest,
	} {
		r := autotests.Timer(mount, callback)
		UnwrapResult(r, aggregate, 0)
	}

	fmt.Printf("\nSummary: %d passed, %d skipped, %d failed, total time %s\n", aggregate.Passed, aggregate.Skipped, aggregate.Failed, aggregate.Duration)

	time.Sleep(50 * time.Millisecond)
}
