package interfaces

type FormatLogger interface {
	Logf(format string, v ...any)
}

type Logger interface {
	Log(v ...any)
}

type ErrorLogger interface {
	Error(v ...any)
}

type ErrorFormatLogger interface {
	Errorf(format string, v ...any)
}

type LoggerCloser interface {
	Close() error
}
