package logging

import "testing"

const (
	ErrorPrefix   = "[ERROR] "
	InfoPrefix    = "[INFO] "
	TracePrefix   = "[TRACE] "
	UntracePrefix = "[UNTRACE] "
	DebugPrefix   = "[DEBUG] "
)

func Errorf(t *testing.T, template string, args ...interface{}) {
	t.Logf(ErrorPrefix+template, args...)
}

func Infof(t *testing.T, template string, args ...interface{}) {
	t.Logf(InfoPrefix+template, args...)
}

func Debugf(t *testing.T, template string, args ...interface{}) {
	t.Logf(DebugPrefix+template, args...)
}

func Tracef(t *testing.T, template string, args ...interface{}) {
	t.Logf(TracePrefix+template, args...)
}

func Untracef(t *testing.T, template string, args ...interface{}) {
	t.Logf(UntracePrefix+template, args...)
}
