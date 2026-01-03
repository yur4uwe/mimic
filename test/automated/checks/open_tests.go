package autochecks

import (
	"fmt"
	"os"
	"path/filepath"

	autotests "github.com/mimic/test/automated/declarations"
)

func CheckOpenFlagsTest(base string) *autotests.AutoTestResult {
	out := &autotests.AutoTestResult{Name: "CheckOpenFlagsTest", Passed: false}
	fbase := filepath.Join(base, "flags.dat")

	defer func() {
		for _, s := range []string{".baseline", ".O_RDONLY", ".O_WRONLY", ".restoreForTrunc", ".trunc", ".create", ".append"} {
			_ = os.Remove(fbase + s)
		}
	}()

	bres := autotests.PrepareFileForTest(fbase+".baseline", autotests.NewFilePreparationOptions().WithData([]byte("BASE")).RequestedBy("Open Flags Baseline"))
	out.Children = append(out.Children, bres)
	if !bres.Passed {
		out.Error = fmt.Errorf("baseline failed: %w", bres.Error)
		return out
	}

	runCheck := func(name, p string, check func(string) error, data []byte, prep bool) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: name, Passed: false}
		if prep {
			var pr *autotests.AutoTestResult
			if data != nil {
				pr = autotests.PrepareFileForTest(p, autotests.NewFilePreparationOptions().WithData(data).RequestedBy(name))
			} else {
				pr = autotests.PrepareFileForTest(p, nil)
			}
			r.Children = append(r.Children, pr)
			if !pr.Passed {
				r.Error = fmt.Errorf("%s prep failed: %w", name, pr.Error)
				return r
			}
		}

		t := autotests.Timer(p, func(path string) *autotests.AutoTestResult {
			tr := &autotests.AutoTestResult{Name: name, Passed: false}
			if err := check(path); err != nil {
				tr.Error = fmt.Errorf("%s failed: %w", name, err)
				return tr
			}
			tr.Passed = true
			return tr
		})
		r.Children = append(r.Children, t)
		if !t.Passed {
			r.Error = t.Error
			_ = os.Remove(p)
			return r
		}
		_ = os.Remove(p)
		r.Passed = true
		return r
	}

	out.Children = append(out.Children, runCheck("O_RDONLY", fbase+".O_RDONLY", checkO_RDONLY_new, []byte("BASE"), true))
	out.Children = append(out.Children, runCheck("O_WRONLY", fbase+".O_WRONLY", checkO_WRONLY_new, []byte("BASE"), true))
	out.Children = append(out.Children, runCheck("O_APPEND", fbase+".append", checkO_APPEND_new, []byte("BASE"), true))
	out.Children = append(out.Children, runCheck("O_TRUNC", fbase+".trunc", checkO_TRUNC_new, []byte("HELLO WORLD"), true))
	_ = os.Remove(fbase + ".create")
	out.Children = append(out.Children, runCheck("O_CREATE", fbase+".create", checkO_CREATE_new, nil, false))

	out.Passed = true
	return out
}
