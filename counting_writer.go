package main

import (
	"sync/atomic"
)

type CountingWriter struct {
	BytesWritten *int64
}

func (w *CountingWriter) Write(p []byte) (n int, err error) {
	atomic.AddInt64(w.BytesWritten, int64(len(p)))
	return len(p), nil
}

func (w *CountingWriter) Close() error {
	return nil
}
