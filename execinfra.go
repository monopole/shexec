package shexec

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
			verboseLoggingEnabled, channeler.VerboseLoggingEnabled =
				p.EnableDetailedLogging, p.EnableDetailedLogging
			//nolint:wrapcheck
			return channeler.Start(&p.Params)
		},
		p.SentinelOut,
		p.SentinelErr,
	)
}

const errCategory = "shexec infra"

func shErr(format string, a ...any) error {
	// nolint:goerr113
	return fmt.Errorf("%s; %s", errCategory, fmt.Sprintf(format, a...))
}

func shErrCaused(err error, format string, a ...any) error {
	return fmt.Errorf("%s; %s; %w", errCategory, fmt.Sprintf(format, a...), err)
}

// channelsMakerF can be mocked in tests with bare channels
// (channels not associated with a shell, just made in a test).
type channelsMakerF func() (*channeler.Channels, error)

// NewShellRaw returns a new Shell in the off state, built from
// the given channels-maker function and the two sentinels.
// Allows testing with injected channels instead of a real shell subprocess.
func NewShellRaw(f channelsMakerF, so Sentinel, se Sentinel) Shell {
	// Uncomment when debugging.
	// verboseLoggingEnabled, channeler.VerboseLoggingEnabled = true, true
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

func (eInf *execInfra) infraStart(d time.Duration) error {
	var err error
	eInf.channels, err = eInf.chMaker()
	if err != nil {
		return shErrCaused(err, "chMaker start failure")
	}
	eInf.chInfraErr = make(chan error)
	if !eInf.haveErrSentinel() {
		// Fire off a thread to drain the stdErr channel
		// so that it doesn't fill up and block the shell.
		// No need for such a drain on stdOut, as we'll
		// always want to parse it normally.
		go func() {
			lgr.Println("infraStart; no err sentinel, will drain stdErr")
			for range eInf.channels.StdErr {
				// just throw it away
			}
		}()
	}
	lgr.Println("infraStart; testing sentinels to make sure they work")
	gotSentinels := eInf.fireOffSentinelFilters(DevNull, DevNull)
	select {
	case <-gotSentinels:
		lgr.Println("infraStart; got sentinels at startup, yay")
		return nil
	case err = <-eInf.chInfraErr:
		lgr.Println("infraStart; got infra error in start call")
		return err
	case <-time.After(d):
		return shErr("starting, but no sentinels found after %s", d)
	}
}

func (eInf *execInfra) infraRun(d time.Duration, c Commander) error {
	if c == nil {
		return shErr("must specify a non-nil commander to Run")
	}
	lgr.Printf("infraRun; starting: %q", c.Command())
	eInf.channels.StdIn <- c.Command()
	lgr.Printf("infraRun; enqueued command %s", abbrev(c.Command()))
	gotSentinels := eInf.fireOffSentinelFilters(c.ParseOut(), c.ParseErr())
	select {
	case <-gotSentinels:
		lgr.Printf(
			"infraRun; got sentinels after command %q", abbrev(c.Command()))
		return nil
	case err := <-eInf.channels.Done:
		lgr.Printf(
			"infraRun; channels.Done ended unexpectedly with err: %s",
			err.Error())
		return err
	case err := <-eInf.chInfraErr:
		lgr.Println("infraRun; got infra error in run call")
		return err
	case <-time.After(d):
		lgr.Printf("infraRun; no sentinels found after %s", d)
		return shErr(
			"running %q, no sentinels found after %s",
			abbrev(c.Command()), d)
	}
}

func (eInf *execInfra) infraStop(d time.Duration, c bareCommand) error {
	if c != "" {
		lgr.Printf("infraStop; sending final command %q to stdin", c)
		eInf.channels.StdIn <- string(c)
		lgr.Printf("infraStop; successfully enqueued stop command %q", c)
	} else {
		lgr.Printf("infraStop; no final command")
		// A possible problem here is that if the last command sent
		// was the error sentinel, then the process will exit with whatever
		// code sits in $?, likely 127 ("command not found").
		// To avoid this, send the error sentinel _before_ the out sentinel.
	}
	close(eInf.channels.StdIn)
	close(eInf.chInfraErr)
	eInf.chInfraErr = nil // Assure that this will block if used in select.
	select {
	case hopefullyNil := <-eInf.channels.Done:
		lgr.Printf("infraStop; signal on Done = %s", hopefullyNil)
		return hopefullyNil
	case <-time.After(d):
		lgr.Printf("infraStop; timeout of %s expired", d)
		return shErr("stop failure; shell not done after %s", d)
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
		lgr.Printf(
			"fire; sending sentinelErr command %q to stdIn", eInf.sentinelErr.C)
		eInf.channels.StdIn <- eInf.sentinelErr.C
		lgr.Printf(
			"fire; successfully enqueued sentinelErr command %q",
			eInf.sentinelErr.C)
		go scanForSentinel(
			eInf.channels.StdErr, "stdErr", &sentinelWait, stdErr,
			eInf.sentinelErr.V, eInf.chInfraErr)
	}

	sentinelWait.Add(1)
	lgr.Printf(
		"fire; sending sentinelOut command %q to stdIn", eInf.sentinelOut.C)
	eInf.channels.StdIn <- eInf.sentinelOut.C
	lgr.Printf(
		"fire; successfully enqueued sentinelOut command %q",
		eInf.sentinelOut.C)
	go scanForSentinel(
		eInf.channels.StdOut, "stdOut", &sentinelWait, stdOut,
		eInf.sentinelOut.V, eInf.chInfraErr)

	gotSentinels := make(chan bool)
	go func() {
		if eInf.haveErrSentinel() {
			lgr.Printf("fire; awaiting both sentinels")
		} else {
			lgr.Printf("fire; awaiting stdOut sentinel")
		}
		sentinelWait.Wait()
		lgr.Printf("fire; done with sentinelWait.Wait")
		gotSentinels <- true
	}()
	return gotSentinels
}

// scanForSentinel is a thread that reads from a channel (stdOut or stdErr)
// and looks for sentinel response values.
// When a line has a sentinel value, the command parser is closed, and
// sentinelWait.Done is called signalling that a sentinel has been acquired
// and the thread ends happily.
// If the line doesn't have a sentinel, it's forwarded to the parser and the
// thread continues scanning and forwarding.
// If the input channel closes without detection of a sentinel value, an error
// is dumped into chErr.
//
//nolint:gocognit
func scanForSentinel(
	stream <-chan string,
	name string,
	sentinelWait *sync.WaitGroup,
	parser io.WriteCloser,
	senValue string,
	chErr chan<- error,
) {
	lgr.Printf("scan %s; awaiting process output", name)
	for line := range stream {
		lgr.Printf("scan %s; got line: %q", name, abbrev(line))
		if p := strings.TrimSuffix(line, senValue); len(p) < len(line) {
			// Sentinel value found at end of line.
			// Stop reading stream and return.
			lgr.Printf(
				"scan %s; matched sentinel %q to end of line", name, senValue)
			if len(p) > 0 {
				// Oops, we have something on the command line *before*
				// the sentinel - send it to the parser as it might be
				// a valid command.
				lgr.Printf("scan %s; writing partial line %q", name, abbrev(p))
				if _, err := parser.Write([]byte(p)); err != nil {
					chErr <- shErrCaused(
						err,
						"problem writing partial %q to %s parser",
						p, name)
					return
				}
			}
			lgr.Printf("scan %s; sentinel in hand, closing", name)
			if err := parser.Close(); err != nil {
				chErr <- shErrCaused(err, "problem (1) closing %s parser", name)
				return
			}
			sentinelWait.Done()
			// This is the happy exit.
			lgr.Printf("scan %s; happily closed", name)
			return
		}
		lgr.Printf(
			"scan %s; forwarding non-sentinel line %q", name, abbrev(line))
		// Pass the data on.
		if _, err := parser.Write([]byte(line)); err != nil {
			chErr <- shErrCaused(
				err, "problem writing line %q to %s parser", abbrev(line), name)
			return
		}
		lgr.Printf("scan %s; awaiting process output", name)
	}
	if err := parser.Close(); err != nil {
		chErr <- shErrCaused(err, "problem (2) closing %s parser", name)
		return
	}
	lgr.Printf("%s closed before sentinel %q found", name, senValue)
	// It's likely that the subprocess crashed/ended on error.
	chErr <- shErr("%s closed before sentinel %q found", name, senValue)
}
