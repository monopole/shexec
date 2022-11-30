package shexec

import "io"

// DevNull is an io.WriteCloser that does nothing.
var DevNull io.WriteCloser = &discard{}

// discard is an io.WriteCloser that does nothing.
type discard struct{}

func (dn *discard) Write(x []byte) (int, error) { return len(x), nil }
func (dn *discard) Close() error                { return nil }
