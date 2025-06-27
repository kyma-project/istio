package logging

import "testing"

const (
	ErrorPrefix = "[ERROR] "
	InfoPrefix  = "[INFO] "
	TracePrefix = "[TRACE] "
	DebugPrefix = "[DEBUG] "
)

func Errorf(t *testing.T, template string, args ...interface{}) {
	t.Helper()
	t.Logf(ErrorPrefix+template, args...)
}

func Infof(t *testing.T, template string, args ...interface{}) {
	t.Helper()
	t.Logf(InfoPrefix+template, args...)
}

func Debugf(t *testing.T, template string, args ...interface{}) {
	t.Helper()
	t.Logf(DebugPrefix+template, args...)
}

func Tracef(t *testing.T, template string, args ...interface{}) {
	t.Helper()
	t.Logf(TracePrefix+template, args...)
}
