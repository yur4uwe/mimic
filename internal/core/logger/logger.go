package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/mimic/internal/interfaces"
)

type FullLogger interface {
	interfaces.Logger
	interfaces.FormatLogger
	interfaces.ErrorLogger
	interfaces.ErrorFormatLogger
}

type Logger struct {
	verbose bool
	w       io.Writer
	files   []*os.File
}

func New(verbose bool, outputs []string) FullLogger {
	var writers []io.Writer
	var files []*os.File

	if len(outputs) == 0 {
		writers = append(writers, os.Stdout)
	} else {
		for _, out := range outputs {
			switch out {
			case "", "stdout":
				writers = append(writers, os.Stdout)
			case "stderr":
				writers = append(writers, os.Stderr)
			default:
				f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "logger: failed to open %q: %v; skipping\n", out, err)
					continue
				}
				writers = append(writers, f)
				files = append(files, f)
			}
		}
	}

	mw := io.MultiWriter(writers...)
	return &Logger{
		verbose: verbose,
		w:       mw,
		files:   files,
	}
}

func (l *Logger) Log(v ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintln(l.w, v...)
}

func (l *Logger) Logf(format string, v ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintf(l.w, format+"\n", v...)
}

func (l *Logger) Error(v ...any) {
	fmt.Fprintln(l.w, v...)
}

func (l *Logger) Errorf(format string, v ...any) {
	fmt.Fprintf(l.w, format+"\n", v...)
}

// Close optionally closes any opened files. Call if you need to flush/close log files.
func (l *Logger) Close() {
	for _, f := range l.files {
		_ = f.Close()
	}
}
