package autochecks

import (
	"os"
	"path/filepath"

	autotests "github.com/mimic/test/automated/declarations"
)

func CheckOpenFlagsTest(base string) *autotests.AutoTestResult {
	out := &autotests.AutoTestResult{Name: "CheckOpenFlagsTest", State: autotests.TestFailed}
	fbase := filepath.Join(base, "flags.dat")

	defer func() {
		for _, s := range []string{".baseline", ".O_RDONLY", ".O_WRONLY", ".restoreForTrunc", ".trunc", ".create", ".append"} {
			_ = os.Remove(fbase + s)
		}
	}()

	bres := autotests.PrepareFileForTest(fbase+".baseline", autotests.NewFilePreparationOptions().WithData([]byte("BASE")).RequestedBy("Open Flags Baseline"))
	if !out.AddChildResultAndContinue(bres) {
		return out
	}

	runCheck := func(name, p string, check func(string) *autotests.AutoTestResult, data []byte) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: name, State: autotests.TestFailed}
		if data != nil {
			pr := autotests.PrepareFileForTest(p, autotests.NewFilePreparationOptions().WithData(data).RequestedBy(name))
			if !r.AddChildResultAndContinue(pr) {
				return r
			}
		}

		t := autotests.Timer(p, check)
		if !r.AddChildResultAndContinue(t) {
			return r
		}
		r.State = autotests.TestPassed
		return r
	}

	out.Children = append(out.Children, runCheck("O_RDONLY", fbase+".O_RDONLY", checkO_RDONLY_new, []byte("BASE")))
	out.Children = append(out.Children, runCheck("O_WRONLY", fbase+".O_WRONLY", checkO_WRONLY_new, []byte("BASE")))
	out.Children = append(out.Children, runCheck("O_APPEND", fbase+".append", checkO_APPEND_new, []byte("BASE")))
	out.Children = append(out.Children, runCheck("O_TRUNC", fbase+".trunc", checkO_TRUNC_new, []byte("HELLO WORLD")))
	out.Children = append(out.Children, runCheck("O_CREATE", fbase+".create", checkO_CREATE_new, nil))
	out.State = autotests.TestPassed
	return out
}
