package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
	verbose   bool
	stdLogger *log.Logger
	errLogger *log.Logger
	files     []*os.File
}

func New(verbose bool, stdlog, errlog string) (FullLogger, error) {
	var std_writer, err_writer io.Writer
	var files []*os.File
	var createErr error

	tryOpenFile := func(path string) (*os.File, error) {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("cannot create log directory: %v", err)
			}
		}
		return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}

	switch stdlog {
	case "stdout", "":
		std_writer = os.Stdout
	case "discard":
		std_writer = io.Discard
	default:
		f, err := tryOpenFile(stdlog)
		if err != nil {
			createErr = fmt.Errorf("cannot open standard log file: %v", err)
			goto failure
		}

		std_writer = f
		files = append(files, f)
	}

	switch errlog {
	case "stderr", "":
		err_writer = os.Stderr
	case "discard":
		err_writer = io.Discard
	default:
		if errlog != stdlog {
			f, err := tryOpenFile(errlog)
			if err != nil {
				createErr = fmt.Errorf("cannot open error log file: %v", err)
				goto failure
			}

			err_writer = f
			files = append(files, f)
		} else {
			err_writer = std_writer
		}
	}

	return &Logger{
		verbose:   verbose,
		stdLogger: log.New(std_writer, "", log.LstdFlags|log.Lmicroseconds),
		errLogger: log.New(err_writer, "", log.LstdFlags|log.Lmicroseconds),
		files:     files,
	}, nil

failure:
	for _, f := range files {
		_ = f.Close()
	}
	return nil, createErr
}

func (l *Logger) Log(v ...any) {
	if !l.verbose {
		return
	}
	if l.stdLogger != nil {
		l.stdLogger.Print(v...)
	}
}

func (l *Logger) Logf(format string, v ...any) {
	if !l.verbose {
		return
	}
	if l.stdLogger != nil {
		l.stdLogger.Printf(format, v...)
	}
}

func (l *Logger) Error(v ...any) {
	if l.errLogger != nil {
		l.errLogger.Print(v...)
	}
}

func (l *Logger) Errorf(format string, v ...any) {
	if l.errLogger != nil {
		l.errLogger.Printf(format, v...)
	}
}

func (l *Logger) Close() error {
	var err error
	for _, f := range l.files {
		err = errors.Join(err, f.Close())
	}
	return err
}
