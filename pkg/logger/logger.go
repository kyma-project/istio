package logger

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

type LogLevel int

const (
	Error = iota
	Warn
	Info
	Debug
	Trace
)

const (
	_error     = "ERROR"
	_warn      = "WARN"
	_info      = "INFO"
	_debug     = "DEBUG"
	_trace     = "TRACE"
	_undefined = "UNDEFINED"
)

func NewLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case _error:
		return Error
	case _warn:
		return Warn
	case _info:
		return Info
	case _debug:
		return Debug
	case _trace:
		return Trace
	default:
		return Info
	}
}

func (l LogLevel) String() string {
	switch l {
	case Error:
		return _error
	case Warn:
		return _warn
	case Info:
		return _info
	case Debug:
		return _debug
	case Trace:
		return _trace
	default:
		return _undefined
	}
}

type Logger struct {
	maxLevel  LogLevel
	callDepth int
	values    []any
}

func NewLogger(maxLevel LogLevel) *Logger {
	return &Logger{maxLevel: maxLevel}
}

func (l Logger) Init(info logr.RuntimeInfo) {
	l.callDepth = info.CallDepth
}

func (l Logger) Enabled(level int) bool {
	return level <= int(l.maxLevel)
}

func (l Logger) Info(level int, msg string, keysAndValues ...any) {
	if level < 0 {
		level = 0
	}
	switch LogLevel(level) {
	case Info:
		l.write(LogLevel(Info).String(), msg, keysAndValues...)
	case Debug:
		l.write(LogLevel(Debug).String(), msg, keysAndValues...)
	case Trace:
		l.write(LogLevel(Trace).String(), msg, keysAndValues...)
	default:
		// do nothing
	}
}

func (l Logger) Error(err error, msg string, keysAndValues ...any) {
	l.write(LogLevel(Error).String(), fmt.Sprintf("%s: %v", msg, err), keysAndValues...)
}

func (l Logger) WithValues(keysAndValues ...any) logr.LogSink {
	l.values = append(l.values, keysAndValues...)
	return &l
}

func (l Logger) WithName(name string) logr.LogSink {
	l.values = append(l.values, "logger", name)
	return &l
}

func (l Logger) write(prefix, args string, keysAndValues ...any) {
	var parts []string
	parts = append(parts, time.Now().Format("2006-01-02T15:04:05Z"))
	if prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, args)

	kvString := formatKeyValues(append(l.values, keysAndValues...))
	if kvString != "" {
		parts = append(parts, kvString)
	}

	fmt.Println(strings.Join(parts, "\t"))
}

func formatKeyValues(keysAndValues []any) string {
	if len(keysAndValues) == 0 {
		return ""
	}

	var buffer bytes.Buffer
	for i := 0; i < len(keysAndValues); i += 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		var value string
		if i+1 < len(keysAndValues) {
			value = fmt.Sprintf("%v", keysAndValues[i+1])
		}
		buffer.WriteString(fmt.Sprintf("%s=%s ", key, value))
	}
	return strings.TrimSpace(buffer.String())
}
