package scripter

// DevNull is an io.WriteCloser that does nothing.
var DevNull = &devNullDevice{}

// devNullDevice is an io.WriteCloser that does nothing.
type devNullDevice struct {
}

func (dn *devNullDevice) Write(_ []byte) (int, error) {
	return 0, nil
}

func (dn *devNullDevice) Close() error {
	return nil
}
