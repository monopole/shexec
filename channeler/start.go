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

	// Start the output scanners.
	// They will close their output channels when the process exits.
	scanWg.Add(1)
	go handleOutput(&scanWg, chDone, p.ChTimeoutOut, chStdOut, "stdOut", scanOut)
	scanWg.Add(1)
	go handleOutput(&scanWg, chDone, p.ChTimeoutOut, chStdErr, "stdErr", scanErr)

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
	defer close(chDone)
	logger.Printf("stdIn; starting scan to forward to subprocess")
	if terminator != 0 {
		logger.Printf("stdIn; command terminator == '%c'", terminator)
	} else {
		logger.Printf("stdIn; no command terminator")
	}
	var line string
	timer := time.NewTimer(timeout)
	stillOpen := true
	for stillOpen {
		if !timer.Stop() {
			logger.Printf("stdIn; sleepy timer draining")
			<-timer.C
			logger.Printf("stdIn; sleepy timer drained")
		}
		timer.Reset(timeout)
		logger.Printf("stdIn; sleepy timer reset")

		select {
		case line, stillOpen = <-chStdIn:
			if stillOpen {
				bytes := assureTermination(line, terminator)
				logger.Printf("stdIn; issuing: %q", string(bytes))
				if _, err := stdIn.Write(bytes); err != nil {
					logger.Printf("stdIn; unable to write stdIn; %s", err.Error())
					chDone <- fmt.Errorf("unable to write to stdIn; %w", err)
					return
				}
			} else {
				logger.Print("stdIn; detected external closure, shutting down!")
				chStdIn = nil
			}
		case <-timer.C:
			logger.Printf("stdIn; sleepy timeout of %s elapsed", timeout)
			logger.Print("stdIn; why is client taking so long to issue another command?")
			logger.Print("stdIn; sending error, abandoning process.")
			chDone <- fmt.Errorf(
				"timeout of %s elapsed awaiting for input or close on stdin",
				timeout)
			return
		}
	}
	logger.Printf("stdIn; channel closed from the outside (presumably on purpose)")
	if err := stdIn.Close(); err != nil {
		logger.Printf("stdIn; unable to close true stdIn")
		chDone <- fmt.Errorf("unable to close stdIn; %w", err)
		return
	}
	// TODO: add timeout on these waits?
	logger.Printf("stdIn; awaiting stdOut and stdErr scanner exit")
	scanWg.Wait()
	if err := cmdWait(); err != nil {
		logger.Printf("cmd.Wait returns error: %s", err.Error())
		chDone <- fmt.Errorf("cmd.Wait returns: %w", err)
		return
	}
	if err := scanOut.Err(); err != nil {
		logger.Printf("stdIn; stdOut scan error: %s", err.Error())
		chDone <- fmt.Errorf("stdout scan incomplete; %w", err)
		return
	}
	if err := scanErr.Err(); err != nil {
		logger.Printf("stdIn; stdErr scan error: %s", err.Error())
		chDone <- fmt.Errorf("stderr scan incomplete; %w", err)
		return
	}
}

func handleOutput(
	wg *sync.WaitGroup,
	chDone chan<- error,
	timeout time.Duration,
	chStream chan<- string,
	name string,
	scanner *bufio.Scanner,
) {
	logger.Printf("%s; scanning...", name)
	count := 0
	timer := time.NewTimer(timeout)
	for scanner.Scan() {
		line := scanner.Text()
		count++
		logger.Printf("%s; read line #%d: %q", name, count, abbrev(line))
		if !timer.Stop() {
			logger.Printf("%s; backpressure timer draining", name)
			<-timer.C
			logger.Printf("%s; backpressure timer drained", name)
		}
		timer.Reset(timeout)
		logger.Printf("%s; backpressure timer reset", name)
		select {
		case chStream <- line:
			logger.Printf("%s; forwarded line", name)
			// Yay, whatever is reading this accepted the output.
		case <-timer.C:
			// Something should drain chStream, even if only to discard
			// the strings to /dev/null.
			// If the stream channel's buffer fills up, this loop
			// over Scan() won't finish, which means a call to
			// cmd.Wait() will block.  Adding a timeout here to help
			// diagnose that particular situation.
			logger.Printf("%s; backpressure timeout of %s elapsed", name, timeout)
			chDone <- fmt.Errorf(
				"timeout of %s elapsed awaiting write to %s",
				timeout, name)
			close(chStream)
			wg.Done()
			return
		}
	}
	close(chStream)
	logger.Printf("%s; successfully consumed %d lines", name, count)
	wg.Done()
}
