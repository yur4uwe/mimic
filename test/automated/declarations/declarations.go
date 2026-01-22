package autotests

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

type TestResultState uint8

const (
	TestPassed TestResultState = iota
	TestFailed
	TestSkipped
	TestChildFailed
)

type AutoTestResult struct {
	Name     string
	State    TestResultState
	Error    error
	Duration time.Duration
	Children []*AutoTestResult
}

func (atr *AutoTestResult) AddChildResultAndContinue(child *AutoTestResult) bool {
	atr.Children = append(atr.Children, child)
	if child.State == TestFailed {
		atr.State = TestChildFailed
		atr.Error = fmt.Errorf("child %s failed: %w", child.Name, child.Error)
		return false
	}
	return true
}

type FilePreparationOptions struct {
	Size      int
	Data      []byte
	Collect   bool
	Requester string
}

func NewFilePreparationOptions() *FilePreparationOptions {
	return &FilePreparationOptions{
		Size:      0,
		Data:      nil,
		Collect:   false,
		Requester: "default",
	}
}

func (fpo *FilePreparationOptions) WithData(data []byte) *FilePreparationOptions {
	if data == nil {
		return fpo
	}
	fpo.Data = data
	fpo.Size = len(data)
	return fpo
}

func (fpo *FilePreparationOptions) ToCollect() *FilePreparationOptions {
	fpo.Collect = true
	return fpo
}

func (fpo *FilePreparationOptions) RequestedBy(name string) *FilePreparationOptions {
	fpo.Requester = name
	return fpo
}

func PrepareFileForTest(path string, optsArr ...*FilePreparationOptions) *AutoTestResult {
	var opts *FilePreparationOptions
	if len(optsArr) == 0 {
		opts = &FilePreparationOptions{
			Size:      0,
			Data:      nil,
			Collect:   false,
			Requester: "default",
		}
	} else {
		opts = optsArr[0]
	}

	res := &AutoTestResult{
		Name:  "PrepareFileForTest for " + opts.Requester,
		State: TestFailed,
	}
	start := time.Now()

	defer func() {
		res.Duration = time.Since(start)
	}()

	fmt.Printf("Preparing file %s with data: %s\n", path, opts.Data)
	if err := os.WriteFile(path, opts.Data, 0644); err != nil {
		res.Error = err
		_ = os.Remove(path)
		return res
	}

	res.State = TestPassed
	return res
}

// Timer runs a test callback with the filesystem mount path, measures total
// wall time and converts panics into failures.
func Timer(mountPath string, testCallback func(string) *AutoTestResult) *AutoTestResult {
	start := time.Now()

	res := &AutoTestResult{
		State: TestFailed,
	}

	defer func() {
		res.Duration = time.Since(start)
		if r := recover(); r != nil {
			res.Error = fmt.Errorf("panic: %vn%s", r, debug.Stack())
		}
	}()

	inner := testCallback(mountPath)
	if inner == nil {
		return res
	}

	res.Name = inner.Name
	res.State = inner.State
	res.Error = inner.Error
	res.Children = inner.Children

	return res
}
