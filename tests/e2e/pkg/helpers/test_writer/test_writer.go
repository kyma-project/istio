package test_writer

import (
	"bytes"
	"testing"
)

type TLogWriter struct {
	t   *testing.T
	buf bytes.Buffer
}

// NewTLogWriter returns a writer that writes to t.Log.
func NewTLogWriter(t *testing.T) *TLogWriter {
	return &TLogWriter{t: t}
}

func (w *TLogWriter) Write(p []byte) (n int, err error) {
	w.t.Helper()
	n, err = w.buf.Write(p)
	if err != nil {
		return n, err
	}

	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.t.Log(line[:len(line)-1])
	}

	return n, nil
}
