package util

import "io"

type CountingWriter struct {
	io.Writer
	BytesWritten int

	Lookback  int
	LastBytes []byte
}

func NewCountingWriter(w io.Writer) *CountingWriter {
	return NewLookbackCountingWriter(w, 0)
}

func NewLookbackCountingWriter(w io.Writer, lookback int) *CountingWriter {
	return &CountingWriter{
		Writer:       w,
		BytesWritten: 0,
		Lookback:     lookback,
		LastBytes:    make([]byte, lookback),
	}
}

func (w *CountingWriter) Write(b []byte) (int, error) {
	n, err := w.Writer.Write(b)

	if err == nil {
		w.BytesWritten += n

		if n > 0 {
			copy(w.LastBytes, b[capneg(n-w.Lookback):n])
		}
	}

	return n, err
}

func capneg(n int) int {
	if n < 0 {
		return 0
	} else {
		return n
	}
}
