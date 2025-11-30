package logger

import (
	"errors"
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
	interfaces.LoggerCloser
}

type Logger struct {
	verbose bool
	stdw    io.Writer
	errw    io.Writer
	files   []*os.File
}

func New(verbose bool, stdlog, errlog string) (FullLogger, error) {
	var std_writer, err_writer io.Writer
	var files []*os.File

	if stdlog == "stdout" || stdlog == "" {
		std_writer = os.Stdout
	} else {
		f, err := os.OpenFile(stdlog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("cannot open log file: %v", err)
		}
		std_writer = f
		files = append(files, f)
	}
	if errlog == "stderr" || errlog == "" {
		err_writer = os.Stderr
	} else {
		f, err := os.OpenFile(errlog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("cannot open error log file: %v", err)
		}
		err_writer = f
		files = append(files, f)
	}

	return &Logger{
		verbose: verbose,
		stdw:    std_writer,
		errw:    err_writer,
		files:   files,
	}, nil
}

func (l *Logger) Log(v ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintln(l.stdw, v...)
}

func (l *Logger) Logf(format string, v ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintf(l.stdw, format+"\n", v...)
}

func (l *Logger) Error(v ...any) {
	fmt.Fprintln(l.errw, v...)
}

func (l *Logger) Errorf(format string, v ...any) {
	fmt.Fprintf(l.errw, format+"\n", v...)
}

func (l *Logger) Close() error {
	var err error
	for _, f := range l.files {
		err = errors.Join(err, f.Close())
	}
	return err
}
