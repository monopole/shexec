package shexec

import (
	"fmt"
	"io"
	"os"
)

// Commander knows a CLI command,
// and knows how to parse the command's output.
type Commander interface {
	// Command is the actual command to issue to the shell.
	Command() string
	// ParseOut will be written with whatever comes out of
	// the shell's stdOut as the result of issuing Command.
	// Close will be called when the shell believes that
	// all output has been obtained.
	ParseOut() io.WriteCloser
	// ParseErr is like ParseOut, except stdErr is used instead of stdOut.
	ParseErr() io.WriteCloser
}

// DiscardCommander discards everything from its parsers.
type DiscardCommander struct {
	C string
}

func (c *DiscardCommander) Command() string          { return c.C }
func (c *DiscardCommander) ParseOut() io.WriteCloser { return DevNull }
func (c *DiscardCommander) ParseErr() io.WriteCloser { return DevNull }

// PassThruCommander forwards data to the current process stdOut and stdErr.
type PassThruCommander struct{ C string }

func (c *PassThruCommander) Command() string          { return c.C }
func (c *PassThruCommander) ParseOut() io.WriteCloser { return os.Stdout }
func (c *PassThruCommander) ParseErr() io.WriteCloser { return os.Stderr }

// LabellingCommander passes subprocess output from stdOut and stdErr
// to the main process' stdOut, adding a prefix to make a distinction.
type LabellingCommander struct {
	C    string
	wOut io.WriteCloser
	wErr io.WriteCloser
}

// NewLabellingCommander returns an instance of LabellingCommander.
func NewLabellingCommander(c string) *LabellingCommander {
	return &LabellingCommander{
		C:    c,
		wOut: &labellingPrinter{"out"},
		wErr: &labellingPrinter{"err"},
	}
}

func (c *LabellingCommander) Command() string          { return c.C }
func (c *LabellingCommander) ParseOut() io.WriteCloser { return c.wOut }
func (c *LabellingCommander) ParseErr() io.WriteCloser { return c.wErr }

type labellingPrinter struct{ prefix string }

func (sp *labellingPrinter) Close() error { return nil }
func (sp *labellingPrinter) Write(data []byte) (int, error) {
	if sp.prefix == "" {
		return fmt.Println(string(data))
	}
	_, err := fmt.Printf("%s: %s\n", sp.prefix, string(data))
	return len(data), err
}

// RecallCommander remembers all the non-empty lines it sees.
type RecallCommander struct {
	C    string
	wOut LineAbsorber
	wErr LineAbsorber
}

// NewRecallCommander returns an instance of RecallCommander.
func NewRecallCommander(c string) *RecallCommander {
	return &RecallCommander{C: c}
}

func (c *RecallCommander) Command() string          { return c.C }
func (c *RecallCommander) ParseOut() io.WriteCloser { return &c.wOut }
func (c *RecallCommander) ParseErr() io.WriteCloser { return &c.wErr }
func (c *RecallCommander) Reset()                   { c.wErr.Reset(); c.wOut.Reset() }
func (c *RecallCommander) DataOut() []string        { return c.wOut.data }
func (c *RecallCommander) DataErr() []string        { return c.wErr.data }

// LineAbsorber remembers all the non-empty lines it sees.
type LineAbsorber struct{ data []string }

func (ab *LineAbsorber) Reset()          { ab.data = nil }
func (ab *LineAbsorber) Lines() []string { return ab.data }
func (ab *LineAbsorber) Close() error    { return nil }
func (ab *LineAbsorber) Write(data []byte) (int, error) {
	if len(data) > 0 {
		ab.data = append(ab.data, string(data))
	}
	return len(data), nil
}
