package scripter_test

import (
	"fmt"
	"io"
	"time"
)

const (
	// timeOutShort is a "short" timeout, for happy cases.
	timeOutShort = 800 * time.Millisecond
	timeOutTiny  = 30 * time.Millisecond
)

func assertErr(err error) {
	if err == nil {
		panic("example failure: expected an error")
	}
}

func assertNoErr(err error) {
	if err != nil {
		panic("example failure: unexpected err: " + err.Error())
	}
}

// printingCommander just passes subprocess output from stdout and stderr
// to the main process' stdout, adding a prefix to make a distinction.
type printingCommander struct {
	Cmd  string
	wOut io.WriteCloser
	wErr io.WriteCloser
}

func newPrintingCommander(c string) *printingCommander {
	return &printingCommander{
		Cmd:  c,
		wOut: &simplePrinter{"out"},
		wErr: &simplePrinter{"err"},
	}
}

func (c *printingCommander) Command() string {
	return c.Cmd
}

func (c *printingCommander) ParseOut() io.WriteCloser {
	return c.wOut
}

func (c *printingCommander) ParseErr() io.WriteCloser {
	return c.wErr
}

type simplePrinter struct {
	prefix string
}

func (sp *simplePrinter) Write(data []byte) (int, error) {
	return fmt.Printf("%s: %s\n", sp.prefix, string(data))
}

func (sp *simplePrinter) Close() error {
	return nil
}

var devNull = &devNullDevice{}

// devNullDevice is an io.WriteCloser that does nothing.
type devNullDevice struct {
}

func (dn *devNullDevice) Write(_ []byte) (int, error) {
	return 0, nil
}

func (dn *devNullDevice) Close() error {
	return nil
}
