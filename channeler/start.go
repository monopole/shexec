package channeler

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Start starts a shell subprocess, and returns all the channels needed
// to interact with and control it.
// To stop the shell, close it's input channel.
func Start(p *Params) (*Channels, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	cmd := exec.Command(p.Path, p.Args...)
	cmd.Dir = p.WorkingDir

	stdIn, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("getting stdIn for %q; %w", p.Path, err)
	}

	var pipe io.ReadCloser

	pipe, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("getting stdOut for %q; %w", p.Path, err)
	}
	scanOut := bufio.NewScanner(pipe)

	pipe, err = cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("getting stdErr for %q; %w", p.Path, err)
	}
	scanErr := bufio.NewScanner(pipe)

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("trying to start %s - %w", p.Path, err)
	}

	// Make all the communication channels.
	chStdIn := make(chan string, p.BuffSizeIn)
	chStdOut := make(chan string, p.BuffSizeOut)
	chStdErr := make(chan string, p.BuffSizeErr)
	chDone := make(chan error)

	// scanWg lives as long as the process.  It's used to
	// assure capture of the process's exit condition
	// (either success or fail).
	var scanWg sync.WaitGroup

	// Start the output scanners. When the sub-process exits,
	// these Go routines will close the respective channels
	// chStdOut and cnStdErr, and call scanWg.Done.
	// These scanners will live as long as there is output
	// coming from the subprocess. If output is coming in,
	// but the infrastructure isn't consuming it for some reason,
	// then this routine will send an error into
	// chDone.  The timeout countdown is reset whenever output
	// from the given pipe is consumed by the given channel.
	scanWg.Add(1)
	go handleOutput(
		&scanWg, chDone, p.InfraConsumerTimeout,
		chStdOut, "stdOut", scanOut)
	scanWg.Add(1)
	go handleOutput(
		&scanWg, chDone, p.InfraConsumerTimeout,
		chStdErr, "stdErr", scanErr)

	// Start the input thread.  It runs until chStdIn is closed.
	go handleInput(
		&scanWg, chDone, p.ChTimeoutIn, chStdIn,
		stdIn, cmd.Wait, scanOut, scanErr, p.CommandTerminator)

	return &Channels{
		StdIn:  chStdIn,
		StdOut: chStdOut,
		StdErr: chStdErr,
		Done:   chDone,
	}, nil
}

// handleInput monitors, drains and closes all inputs.
// Regrettably it has a high cognitive complexity score.
//
//nolint:gocognit
func handleInput(
	scanWg *sync.WaitGroup,
	chDone chan<- error,
	timeout time.Duration,
	chStdIn <-chan string,
	stdIn io.WriteCloser,
	cmdWait func() error,
	scanOut *bufio.Scanner,
	scanErr *bufio.Scanner,
	terminator byte,
) {
	const name = " stdIn"
	defer close(chDone)
	logger.Printf("%s; starting scan to forward to subprocess", name)
	if terminator != 0 {
		logger.Printf("%s; command terminator == '%c'", name, terminator)
	} else {
		logger.Printf("%s; no command terminator", name)
	}
	var line string
	timer := time.NewTimer(timeout)
	stillOpen := true
	for stillOpen {
		if !timer.Stop() {
			logger.Printf("%s; sleepy timer draining", name)
			<-timer.C
			logger.Printf("%s; sleepy timer drained", name)
		}
		timer.Reset(timeout)
		logger.Printf("%s; sleepy timer reset", name)
		logger.Printf("%s; awaiting command", name)

		select {
		case line, stillOpen = <-chStdIn:
			if stillOpen {
				bytes := assureTermination(line, terminator)
				logger.Printf("%s; issuing: %q", name, string(bytes))
				if _, err := stdIn.Write(bytes); err != nil {
					logger.Printf(
						"%s; unable to write stdIn; %s", name, err.Error())
					chDone <- fmt.Errorf("unable to write to stdIn; %w", err)
					return
				}
			} else {
				logger.Printf(
					"%s; detected external closure, shutting down!", name)
				chStdIn = nil
			}
		case <-timer.C:
			logger.Printf("%s; sleepy timeout of %s elapsed", name, timeout)
			logger.Printf("%s; taking too long to issue another command", name)
			logger.Printf("%s; sending error, abandoning process.", name)
			chDone <- paramErr(
				"timeout of %s elapsed awaiting for input or close on stdin",
				timeout)
			return
		}
	}
	logger.Printf(
		"%s; channel closed from the outside (presumably on purpose)", name)
	if err := stdIn.Close(); err != nil {
		logger.Printf("%s; unable to close true stdIn", name)
		chDone <- fmt.Errorf("unable to close stdIn; %w", err)
		return
	}
	// TODO: add timeout on these waits?
	logger.Printf("%s; awaiting stdOut and stdErr scanner exit", name)
	scanWg.Wait()
	if err := cmdWait(); err != nil {
		logger.Printf("%s; cmd.Wait returns error: %s", name, err.Error())
		chDone <- fmt.Errorf("cmd.Wait returns: %w", err)
		return
	}
	if err := scanOut.Err(); err != nil {
		logger.Printf("%s; stdOut scan error: %s", name, err.Error())
		chDone <- fmt.Errorf("stdout scan incomplete; %w", err)
		return
	}
	if err := scanErr.Err(); err != nil {
		logger.Printf("%s; stdErr scan error: %s", name, err.Error())
		chDone <- fmt.Errorf("stderr scan incomplete; %w", err)
		return
	}
}

func handleOutput(
	wg *sync.WaitGroup,
	chDone chan<- error,
	consumerTimeout time.Duration,
	chStream chan<- string,
	name string,
	scanner *bufio.Scanner,
) {
	logger.Printf("%s; awaiting data from subprocess...", name)
	count := 0
	timer := time.NewTimer(consumerTimeout)
	for scanner.Scan() {
		line := scanner.Text()
		count++
		logger.Printf("%s; have read line #%d: %q", name, count, abbrev(line))
		if !timer.Stop() {
			logger.Printf("%s; backpressure timer draining", name)
			<-timer.C
			logger.Printf("%s; backpressure timer drained", name)
		}
		timer.Reset(consumerTimeout)
		logger.Printf("%s; backpressure timer reset", name)
		select {
		case chStream <- line:
			logger.Printf("%s; forwarded line", name)
			// Yay, the infrastructure processing the subprocess' output
			// is alive and reading this channel.
		case <-timer.C:
			// Subprocess output isn't being consumed fast enough.
			// Something should drain chStream, even if only to discard
			// the strings to /dev/null.
			// If the stream channel's buffer fills up, this loop
			// over Scan() won't finish, which means a call to
			// cmd.Wait() will block. This is the exit hatch to
			// that particular deadlock.
			logger.Printf(
				"%s; backpressure consumerTimeout=%s elapsed",
				name, consumerTimeout)
			chDone <- paramErr(
				"consumerTimeout=%s elapsed awaiting consumer on chan %s",
				consumerTimeout, name)
			close(chStream)
			wg.Done()
			return
		}
		logger.Printf("%s; awaiting data from subprocess...", name)
	}
	logger.Printf("%s; scan done; closing forwarding channel", name)
	close(chStream)
	logger.Printf("%s; successfully consumed %d lines", name, count)
	wg.Done()
}
