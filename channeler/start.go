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
func Start(p *ChParams) (*Channels, error) {
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
	logger.Printf("starting scan of stdIn to forward to subprocess")
	if terminator != 0 {
		logger.Printf("command terminator == '%c'", terminator)
	} else {
		logger.Printf("no command terminator")
	}
	var line string
	timer := time.NewTimer(timeout)
	stillOpen := true
	for stillOpen {
		if !timer.Stop() {
			logger.Printf("input timer draining")
			<-timer.C
			logger.Printf("input timer drained")
		}
		timer.Reset(timeout)
		logger.Printf("input timer reset")

		select {
		case line, stillOpen = <-chStdIn:
			if stillOpen {
				bytes := assureTermination(line, terminator)
				logger.Printf("to stdIn issuing command %q", string(bytes))
				_, err := stdIn.Write(bytes)
				if err != nil {
					chDone <- fmt.Errorf("unable to write to stdIn; %w", err)
					return
				}
			} else {
				logger.Print("detected external closure of stdIn; closing down!")
				chStdIn = nil
			}
		case <-timer.C:
			logger.Printf("timeout of %s elapsed awaiting stdIn", timeout)
			logger.Print("why is client taking so long to issue another command? sending error, abandoning stdIn.")
			chDone <- fmt.Errorf(
				"timeout of %s elapsed awaiting for input or close on stdin",
				timeout)
			return
		}
	}
	logger.Printf("the stdIn channel was closed from the outside")
	if err := stdIn.Close(); err != nil {
		chDone <- fmt.Errorf("unable to close stdIn; %w", err)
		return
	}
	// TODO: add timeout on these waits?
	scanWg.Wait()
	if err := cmdWait(); err != nil {
		logger.Printf("cmd.Wait returns: %s", err.Error())
		chDone <- fmt.Errorf("cmd.Wait returns: %w", err)
		return
	}
	if err := scanOut.Err(); err != nil {
		chDone <- fmt.Errorf("stdout scan incomplete; %w", err)
		return
	}
	if err := scanErr.Err(); err != nil {
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
	logger.Printf("scanning %s...", name)
	count := 0
	timer := time.NewTimer(timeout)
	for scanner.Scan() {
		line := scanner.Text()
		count++
		logger.Printf("from %s read line #%d: %q", name, count, line)
		if !timer.Stop() {
			logger.Printf("%s timer draining", name)
			<-timer.C
			logger.Printf("%s timer drained", name)
		}
		timer.Reset(timeout)
		logger.Printf("%s timer reset", name)
		select {
		case chStream <- line:
			// Yay, whatever is reading this accepted the output.
		case <-timer.C:
			// Something should drain chStream, even if only to discard
			// the strings to /dev/null.
			// If the stream channel's buffer fills up, this loop
			// over Scan() won't finish, which means a call to
			// cmd.Wait() will block.  Adding a timeout here to help
			// diagnose that particular situation.
			chDone <- fmt.Errorf(
				"timeout of %s elapsed awaiting write to %s",
				timeout, name)
			close(chStream)
			wg.Done()
			return
		}
	}
	close(chStream)
	logger.Printf("successfully consumed %d lines from %s.", count, name)
	wg.Done()
}
