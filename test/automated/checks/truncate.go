package autochecks

import (
	"fmt"
	"os"
	"path/filepath"

	autotests "github.com/mimic/test/automated/declarations"
)

func CheckTruncateTest(base string) (res *autotests.AutoTestResult) {
	res = &autotests.AutoTestResult{
		Name:  "CheckTruncateTest",
		State: autotests.TestFailed,
	}

	baseFName := filepath.Join(base, "tfile")

	defer func() {
		_ = os.RemoveAll(baseFName + ".down")
		_ = os.RemoveAll(baseFName + ".up")
	}()

	truncDownRes := autotests.Timer(baseFName+".down", truncateDownTest)
	res.Children = append(res.Children, truncDownRes)
	if truncDownRes.State == autotests.TestFailed {
		res.Error = fmt.Errorf("truncate down test failed: %w", truncDownRes.Error)
		return
	}

	truncUpRes := autotests.Timer(baseFName+".up", truncateUpTest)
	res.Children = append(res.Children, truncUpRes)
	if truncUpRes.State == autotests.TestFailed {
		res.Error = fmt.Errorf("truncate up test failed: %w", truncUpRes.Error)
		return
	}
	res.State = autotests.TestPassed

	return
}

func truncateDownTest(fname string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  "Truncate Down Test",
		State: autotests.TestFailed,
	}

	prep := autotests.PrepareFileForTest(fname, autotests.NewFilePreparationOptions().WithData([]byte("0123456789")).RequestedBy("Truncate Down"))
	res.Children = append(res.Children, prep)
	if prep.State == autotests.TestFailed {
		return res
	}

	if err := os.Truncate(fname, 4); err != nil {
		res.Error = fmt.Errorf("failed: %w", err)
		return res
	}

	b, err := os.ReadFile(fname)
	if err != nil {
		res.Error = fmt.Errorf("post-op read failed: %w", err)
		return res
	}

	if len(b) != 4 {
		res.Error = fmt.Errorf("wrong size: %d should be: 4", len(b))
		return res
	}

	res.State = autotests.TestPassed
	return res
}

func truncateUpTest(fname string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{
		Name:  "Truncate Up Test",
		State: autotests.TestFailed,
	}
	prep := autotests.PrepareFileForTest(fname, autotests.NewFilePreparationOptions().WithData([]byte("0123456789")).RequestedBy("Truncate Down"))
	res.Children = append(res.Children, prep)
	if prep.State == autotests.TestFailed {
		return res
	}
	if err := os.Truncate(fname, 16); err != nil {
		res.Error = fmt.Errorf("failed: %w", err)
		return res
	}
	b, err := os.ReadFile(fname)
	if err != nil {
		res.Error = fmt.Errorf("post-op read failed: %w", err)
		return res
	}
	if len(b) != 16 {
		res.Error = fmt.Errorf("wrong size: %d should be: 16", len(b))
		return res
	}
	res.State = autotests.TestPassed
	return res
}
