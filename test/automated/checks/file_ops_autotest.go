package autochecks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	autotests "github.com/mimic/test/automated/declarations"
)

// CheckFileOpsTest runs file operations as AutoTestResult-style subtests.
func CheckFileOpsTest(base string) *autotests.AutoTestResult {
	res := &autotests.AutoTestResult{Name: "CheckFileOpsTest", Passed: false}

	f := path.Join(base, "basic.create")
	dir := path.Join(base, "test_dir")
	renamed := path.Join(base, "basic.renamed")

	defer func() {
		_ = os.RemoveAll(f)
		_ = os.RemoveAll(renamed)
		_ = os.RemoveAll(dir)
	}()

	// createAndWrite
	c := autotests.Timer(f, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "createAndWrite", Passed: false}
		if err := os.WriteFile(p, []byte("hello world\n"), 0644); err != nil {
			r.Error = fmt.Errorf("createAndWrite: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, c)
	if !c.Passed {
		res.Error = c.Error
		return res
	}

	// appendLine
	a := autotests.Timer(f, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "appendLine", Passed: false}
		fh, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			r.Error = fmt.Errorf("append open: %w", err)
			return r
		}
		if _, err := fh.Write([]byte("append-line\n")); err != nil {
			_ = fh.Close()
			r.Error = fmt.Errorf("append write: %w", err)
			return r
		}
		_ = fh.Close()
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, a)
	if !a.Passed {
		res.Error = a.Error
		return res
	}

	// rename after append
	rnAppend := autotests.Timer(f, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "renameAfterAppend", Passed: false}
		newName := p + ".append"
		if err := os.Rename(p, newName); err != nil {
			r.Error = fmt.Errorf("rename after append: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, rnAppend)
	if !rnAppend.Passed {
		res.Error = rnAppend.Error
		return res
	}
	f = f + ".append"

	// statCheck
	s := autotests.Timer(f, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "statCheck", Passed: false}
		if _, err := os.Stat(p); err != nil {
			r.Error = fmt.Errorf("statCheck: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, s)
	if !s.Passed {
		res.Error = s.Error
		return res
	}

	// makeDir
	m := autotests.Timer(dir, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "makeDir", Passed: false}
		if err := os.Mkdir(p, 0755); err != nil {
			r.Error = fmt.Errorf("makeDir: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, m)
	if !m.Passed {
		res.Error = m.Error
		return res
	}

	// moveIntoDir
	mv := autotests.Timer(dir, func(_ string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "moveIntoDir", Passed: false}
		dest := filepath.Join(dir, filepath.Base(f))
		if err := os.Rename(f, dest); err != nil {
			r.Error = fmt.Errorf("moveIntoDir: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, mv)
	if !mv.Passed {
		res.Error = mv.Error
		return res
	}

	// renameOut
	rn := autotests.Timer(renamed, func(_ string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "renameOut", Passed: false}
		src := filepath.Join(dir, filepath.Base(f))
		if err := os.Rename(src, renamed); err != nil {
			r.Error = fmt.Errorf("renameOut: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, rn)
	if !rn.Passed {
		res.Error = rn.Error
		return res
	}

	// cleanup remove
	cl := autotests.Timer(renamed, func(p string) *autotests.AutoTestResult {
		r := &autotests.AutoTestResult{Name: "cleanupRemove", Passed: false}
		if err := os.Remove(p); err != nil {
			r.Error = fmt.Errorf("cleanup remove: %w", err)
			return r
		}
		r.Passed = true
		return r
	})
	res.Children = append(res.Children, cl)
	if !cl.Passed {
		res.Error = cl.Error
		return res
	}

	res.Passed = true
	return res
}
