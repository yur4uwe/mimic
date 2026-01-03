package autotests

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

type AutoTestResult struct {
	Name     string
	Passed   bool
	Error    error
	Duration time.Duration
	Children []*AutoTestResult
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
		Name:   "PrepareFileForTest for " + opts.Requester,
		Passed: true,
	}
	start := time.Now()

	defer func() {
		res.Duration = time.Since(start)
	}()

	if err := os.WriteFile(path, opts.Data, 0644); err != nil {
		res.Passed = false
		res.Error = err
		_ = os.Remove(path)
		return res
	}

	return res
}

// Timer runs a test callback with the filesystem mount path, measures total
// wall time and converts panics into failures.
func Timer(mountPath string, testCallback func(string) *AutoTestResult) *AutoTestResult {
	start := time.Now()

	res := &AutoTestResult{
		Passed: true,
	}

	defer func() {
		res.Duration = time.Since(start)
		if r := recover(); r != nil {
			res.Passed = false
			res.Error = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()

	inner := testCallback(mountPath)
	if inner == nil {
		return res
	}

	res.Name = inner.Name
	res.Passed = inner.Passed
	res.Error = inner.Error
	res.Children = inner.Children

	return res
}
