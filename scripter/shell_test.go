package scripter_test

import (
	"io"
	"testing"

	"github.com/monopole/shexec/channeler"
	. "github.com/monopole/shexec/scripter"
	"github.com/stretchr/testify/assert"
)

var commandStatus = newPrintingCommander("status")

func makeConchParams() Parameters {
	return Parameters{
		ChParams: channeler.ChParams{
			WorkingDir: "../conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				"--disable-prompt",
			}},
		SentinelOut: Sentinel{
			C: "echo " + unlikelyWord,
			V: unlikelyWord,
		},
		SentinelErr: Sentinel{
			C: unlikelyWord,
			V: `unrecognized command: "` + unlikelyWord + `"`,
		},
	}
}

func TestShellBadPath(t *testing.T) {
	sh := NewShell(Parameters{ChParams: channeler.ChParams{Path: "beamMeUpScotty"}})
	if err := sh.Start(timeOutShort); assert.Error(t, err) {
		assert.Contains(t, err.Error(), `path "beamMeUpScotty" not available; exit status 127`)
	}
}

func TestShellNoSentinelOut(t *testing.T) {
	sh := NewShell(Parameters{
		ChParams: channeler.ChParams{
			Path: "go",
		}})
	err := sh.Start(timeOutShort)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `problem in SentinelOut; must specify Sentinel command`)
}

func TestShellNoSentinelOutValue(t *testing.T) {
	sh := NewShell(Parameters{
		ChParams:    channeler.ChParams{Path: "go"},
		SentinelOut: Sentinel{C: "echo " + unlikelyWord},
	})
	if err := sh.Start(timeOutShort); assert.Error(t, err) {
		assert.Contains(t, err.Error(),
			`problem in SentinelOut; sentinel value "" too short at len=0;`)
	}
}

func TestShellHappy(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Run(timeOutShort, commandStatus))
	assert.NoError(t, sh.Stop(timeOutShort, "exit 0"))
}

func TestShellEmptyStopCommand(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Run(timeOutShort, commandStatus))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
}

func TestShellForgotCommander(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	if err := sh.Run(timeOutShort, nil); assert.Error(t, err) {
		assert.Contains(t, err.Error(), `must specify a non-nil commander to Run`)
	}
}

func TestShellRunWithoutStart(t *testing.T) {
	sh := NewShell(makeConchParams())
	err := sh.Run(timeOutShort, commandStatus)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run called, but shell not started yet")
}

func TestShellStopWithoutStart(t *testing.T) {
	sh := NewShell(makeConchParams())
	if err := sh.Stop(timeOutShort, ""); assert.Error(t, err) {
		assert.Contains(t, err.Error(), "stop called, but shell not started yet")
	}
}

func TestShellStopAfterStop(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Run(timeOutShort, commandStatus))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
	if err := sh.Stop(timeOutShort, ""); assert.Error(t, err) {
		assert.Contains(t, err.Error(), "stop called, but shell not started yet")
	}
}

func TestShellRunAfterStop(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
	if err := sh.Run(timeOutShort, commandStatus); assert.Error(t, err) {
		assert.Contains(t, err.Error(), "run called, but shell not started yet")
	}
}

func TestShellStartStopStart(t *testing.T) {
	sh := NewShell(makeConchParams())
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
	assert.NoError(t, sh.Start(timeOutShort))
	assert.NoError(t, sh.Run(timeOutShort, commandStatus))
	assert.NoError(t, sh.Stop(timeOutShort, ""))
}

// The tests below are white-box tests that don't use a live shell.
// They instead provide artificial channel traffic.

const (
	rawSentOutC = "whateverOut"
	rawSentOutV = "blah boo boo"
	rawSentErrC = "whateverErr"
	rawSentErrV = "lorem ipsum"
	rawCommand  = "avast"
	rawExit     = "that's all folks"
)

func rawSetUp() (chan string, chan error, chan string, chan string, Shell) {
	// Intentionally use no-buffer channels,
	// so that all traffic is accounted for.
	chStdIn := make(chan string)
	chDone := make(chan error)
	chStdOut := make(chan string)
	chStdErr := make(chan string)
	// Return the bare channels for manipulation in test.
	return chStdIn, chDone, chStdOut, chStdErr,
		NewShellRaw(
			func() (*channeler.Channels, error) {
				return &channeler.Channels{
					StdIn:  chStdIn,
					Done:   chDone,
					StdOut: chStdOut,
					StdErr: chStdErr,
				}, nil
			},
			Sentinel{C: rawSentOutC, V: rawSentOutV},
			Sentinel{C: rawSentErrC, V: rawSentErrV},
		)
}

func TestShellRaw1(t *testing.T) {
	chStdIn, chDone, chStdOut, chStdErr, sh := rawSetUp()
	// This thread emulates draining and executing commands
	// from stdIn, and closing chDone when stdIn closes.
	go func() {
		assert.Equal(t, rawSentErrC, <-chStdIn)
		assert.Equal(t, rawSentOutC, <-chStdIn)
		assert.Equal(t, rawExit, <-chStdIn)
		// This blocks until the Stop call below.
		_, stillOpen := <-chStdIn
		assert.False(t, stillOpen)
		// Indicate that nothing further will come in on stdErr or stdOut
		close(chDone)
	}()
	// Send the expected sentinel values.
	go func() { chStdErr <- rawSentErrV }()
	go func() { chStdOut <- rawSentOutV }()
	// Start it.
	assert.NoError(t, sh.Start(timeOutTiny))
	// Stop it.
	assert.NoError(t, sh.Stop(timeOutTiny, rawExit))
}

func TestShellRaw2(t *testing.T) {
	chStdIn, chDone, _, chStdErr, sh := rawSetUp()
	go func() {
		assert.Equal(t, rawSentErrC, <-chStdIn)
		assert.Equal(t, rawSentOutC, <-chStdIn)
		assert.Equal(t, rawExit, <-chStdIn)
		_, stillOpen := <-chStdIn
		assert.False(t, stillOpen)
		close(chDone)
	}()
	go func() { chStdErr <- rawSentErrV }()
	// Send nothing on stdOut (no sentinel value).
	err := sh.Start(timeOutTiny)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no sentinels found after 30ms")
	assert.Error(t, sh.Stop(timeOutTiny, rawExit))
}

type sillyCommand struct{}

func (x *sillyCommand) Command() string          { return rawCommand }
func (x *sillyCommand) ParseOut() io.WriteCloser { return devNull }
func (x *sillyCommand) ParseErr() io.WriteCloser { return devNull }

func TestShellRaw3(t *testing.T) {
	chStdIn, chDone, chStdOut, chStdErr, sh := rawSetUp()
	go func() {
		assert.Equal(t, rawSentErrC, <-chStdIn)
		assert.Equal(t, rawSentOutC, <-chStdIn)
		assert.Equal(t, rawCommand, <-chStdIn)
		assert.Equal(t, rawSentErrC, <-chStdIn)
		assert.Equal(t, rawSentOutC, <-chStdIn)
		assert.Equal(t, rawExit, <-chStdIn)
		_, stillOpen := <-chStdIn
		assert.False(t, stillOpen)
		close(chDone)
	}()
	go func() {
		chStdErr <- rawSentErrV
		chStdErr <- rawSentErrV
	}()
	go func() {
		chStdOut <- rawSentOutV
		chStdOut <- rawSentOutV
	}()
	assert.NoError(t, sh.Start(timeOutTiny))
	assert.NoError(t, sh.Run(timeOutTiny, &sillyCommand{}))
	assert.NoError(t, sh.Stop(timeOutTiny, rawExit))
}
