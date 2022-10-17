package util

import "io"

type CountingWriter struct {
	io.Writer
	BytesWritten int
}

func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		Writer:       w,
		BytesWritten: 0,
	}
}

func (w *CountingWriter) Write(b []byte) (int, error) {
	n, err := w.Writer.Write(b)
	w.BytesWritten += n
	return n, err
}
