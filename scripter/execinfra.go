package scripter

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/monopole/shexec/channeler"
)

// NewShell returns a new Shell built from Parameters in the off state.
func NewShell(p Parameters) Shell {
	return NewShellRaw(
		func() (*channeler.Channels, error) {
			if err := p.Validate(); err != nil {
				return nil, err
			}
			return channeler.Start(&p.ChParams)
		},
		p.SentinelOut,
		p.SentinelErr,
	)
}

// channelsMakerF can be mocked in tests with bare channels
// (channels not associated with a shell, just made in a test).
type channelsMakerF func() (*channeler.Channels, error)

// NewShellRaw returns a new Shell in the off state, built from
// the given channels-maker function and the two sentinels.
// Allows testing with injected channels instead of a real shell subprocess.
func NewShellRaw(f channelsMakerF, so Sentinel, se Sentinel) Shell {
	return &execMutex{
		state: &execStateOff{
			infra: &execInfra{
				chMaker:     f,
				sentinelOut: &so,
				sentinelErr: &se,
			},
		},
	}
}

// execInfra holds Shell infrastructure shared by all Shell states.
type execInfra struct {
	// sentinelOut holds the stdOut sentinel.
	sentinelOut *Sentinel

	// sentinelOut holds the stdErr sentinel.
	sentinelErr *Sentinel

	// chMaker is used to make a fresh set of channels on Start.
	chMaker channelsMakerF

	// channels holds all the pipes in and out of the shell.
	channels *channeler.Channels

	// chInfraErr is used by any internal infrastructure
	// thread to signal a fatal error that requires a
	// restart of the subprocess.
	chInfraErr chan error
}

func (eInf *execInfra) infraStart(d time.Duration) (err error) {
	eInf.channels, err = eInf.chMaker()
	if err != nil {
		return err
	}
	eInf.chInfraErr = make(chan error)
	if !eInf.haveErrSentinel() {
		// Fire off a thread to drain the stdErr channel
		// so that it doesn't fill up and block the shell.
		// No need for such a drain on stdOut, as we'll
		// always want to parse it normally.
		go func() {
			logger.Println("no error sentinel; beginning drain of stdErr")
			for range eInf.channels.StdErr {
			}
		}()
	}
	logger.Println("testing sentinels to make sure they work")
	gotSentinels := eInf.fireOffSentinelFilters(DevNull, DevNull)
	select {
	case <-gotSentinels:
		logger.Println("got sentinels at startup, yay")
		return nil
	case err = <-eInf.chInfraErr:
		logger.Println("got infra error in start call")
		return err
	case <-time.After(d):
		return fmt.Errorf("starting, but no sentinels found after %s", d)
	}
}

func (eInf *execInfra) infraRun(d time.Duration, c Commander) error {
	if c == nil {
		return fmt.Errorf("must specify a non-nil commander to Run")
	}
	logger.Printf("starting run - sending: %q", c.Command())
	eInf.channels.StdIn <- c.Command()
	logger.Printf(
		"successfully enqueued command %q", c.Command())
	gotSentinels := eInf.fireOffSentinelFilters(c.ParseOut(), c.ParseErr())
	select {
	case <-gotSentinels:
		logger.Printf("got sentinels after command %q", c.Command())
		return nil
	case err := <-eInf.chInfraErr:
		logger.Println("got infra error in run call")
		return err
	case <-time.After(d):
		return fmt.Errorf(
			"running %q, no sentinels found after %s", c.Command(), d)
	}
}

func (eInf *execInfra) infraStop(d time.Duration, c bareCommand) error {
	if c != "" {
		logger.Printf("in stop, sending final command %q to stdin", c)
		eInf.channels.StdIn <- string(c)
		logger.Printf(
			"successfully enqueued stop command %q", c)
	} else {
		logger.Printf("in stop, no final command")
		// TODO: a possible problem here is that if the last command sent
		// was the error sentinel, then the process will exit with whatever
		// code sits in $?, likely 127 ("command not found").
		// To avoid this, send the error sentinel before the out sentinel.
	}
	close(eInf.channels.StdIn)
	close(eInf.chInfraErr)
	eInf.chInfraErr = nil // Assure that this will block if used in select.
	select {
	case possibleErr := <-eInf.channels.Done:
		return possibleErr
	case <-time.After(d):
		return fmt.Errorf("shell not done after %s", d)
	}
}

func (eInf *execInfra) haveErrSentinel() bool {
	return eInf.sentinelErr.C != ""
}

// fireOffSentinelFilters sends in the sentinel commands and scans
// the two output streams for sentinel values, passing everything
// that is not a sentinel value to the two respective parsers.
// When both sentinels are found, a signal is sent on the returned channel.
func (eInf *execInfra) fireOffSentinelFilters(
	stdOut, stdErr io.WriteCloser) <-chan bool {

	var sentinelWait sync.WaitGroup

	if eInf.haveErrSentinel() {
		sentinelWait.Add(1)
		logger.Printf(
			"sending sentinelErr command %q to stdIn", eInf.sentinelErr.C)
		eInf.channels.StdIn <- eInf.sentinelErr.C
		logger.Printf(
			"successfully enqueued sentinelErr command %q", eInf.sentinelErr.C)
		go scanForSentinel(
			eInf.channels.StdErr, "stdErr", &sentinelWait, stdErr,
			eInf.sentinelErr.V, eInf.chInfraErr)
	}

	sentinelWait.Add(1)
	logger.Printf(
		"sending sentinelOut command %q to stdIn", eInf.sentinelOut.C)
	eInf.channels.StdIn <- eInf.sentinelOut.C
	logger.Printf(
		"successfully enqueued sentinelOut command %q", eInf.sentinelOut.C)
	go scanForSentinel(
		eInf.channels.StdOut, "stdOut", &sentinelWait, stdOut,
		eInf.sentinelOut.V, eInf.chInfraErr)

	gotSentinels := make(chan bool)
	go func() {
		if eInf.haveErrSentinel() {
			logger.Printf("awaiting both sentinels")
		} else {
			logger.Printf("awaiting stdOut sentinel")
		}
		sentinelWait.Wait()
		logger.Printf("done with sentinelWait.Wait")
		gotSentinels <- true
	}()
	return gotSentinels
}

func scanForSentinel(
	stream <-chan string,
	name string,
	sentWaiter *sync.WaitGroup,
	parser io.WriteCloser,
	senValue string,
	chErr chan<- error,
) {
	logger.Printf("beginning scan of %s for sentinel value %q", name, senValue)
	for line := range stream {
		logger.Printf("In %s stream got line: %q", name, line)
		if p := strings.TrimSuffix(line, senValue); len(p) < len(line) {
			// Sentinel value found, so immediately stop reading stream.
			// If the sentinel value is empty, this block never
			// executes, so the stream will be continually consumed,
			// which would be bad.
			logger.Printf(
				"In %s stream matched sentinel value %q to end of line %q",
				name, senValue, line)
			if len(p) > 0 {
				if _, err := parser.Write([]byte(p)); err != nil {
					chErr <- fmt.Errorf(
						"problem writing partial %q to %s parser; %w", p, name, err)
					return
				}
			}
			logger.Printf("sentinel in hand, closing %s parser", name)
			if err := parser.Close(); err != nil {
				chErr <- fmt.Errorf("problem closing %s parser; %w", name, err)
				return
			}
			sentWaiter.Done()
			// This is the happy exit.
			logger.Printf("happily exiting %s stream scanner", name)
			return
		}
		logger.Printf(
			"In %s stream, no sentinel value %q ending line %q",
			name, senValue, line)
		// Pass the data on.
		if _, err := parser.Write([]byte(line)); err != nil {
			chErr <- fmt.Errorf(
				"problem writing line %q to %s parser; %w", line, name, err)
			return
		}
	}
	logger.Printf("stream %s ended too soon", name)
	// Stream ended too soon. This is the unhappy exit.
	// It's likely that the subprocess crashed.
	chErr <- fmt.Errorf("%s closed before sentinel %q found", name, senValue)
}
