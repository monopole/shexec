package channeler_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/monopole/shexec/channeler"
	"github.com/stretchr/testify/assert"
)

const theShell = "/bin/sh"

func consumeChannel(name string, ch <-chan string) {
	for line := range ch {
		fmt.Printf("%s: %q\n", name, line)
	}
}

func TestStartHappy(t *testing.T) {
	chs, err := Start(&Params{
		Path: theShell,
	})
	assert.NoError(t, err)
	chs.StdIn <- "ls -las /proc/version"
	chs.StdIn <- "more /nonexistent"
	chs.StdIn <- "more /proc/version"
	close(chs.StdIn)
	go consumeChannel("err", chs.StdErr)
	go consumeChannel("out", chs.StdOut)
	assert.NoError(t, <-chs.Done)
}

func TestStartExitZero(t *testing.T) {
	chs, err := Start(&Params{
		Path: theShell,
	})
	assert.NoError(t, err)
	chs.StdIn <- "ls -las /proc/version"
	chs.StdIn <- "exit 0"
	close(chs.StdIn)
	go consumeChannel("err", chs.StdErr)
	go consumeChannel("out", chs.StdOut)
	assert.NoError(t, <-chs.Done)
}

func TestStartExitOne(t *testing.T) {
	chs, err := Start(&Params{
		Path: theShell,
	})
	assert.NoError(t, err)
	chs.StdIn <- "ls -las /proc/version"
	chs.StdIn <- "exit 77"
	close(chs.StdIn)
	go consumeChannel("err", chs.StdErr)
	go consumeChannel("out", chs.StdOut)
	if err = <-chs.Done; assert.Error(t, err) {
		assert.Contains(t, err.Error(), "exit status 77")
	}
}

func TestStartStallOnStdIn(t *testing.T) {
	p := &Params{
		Path:        theShell,
		ChTimeoutIn: 50 * time.Millisecond,
	}
	chs, err := Start(p)
	assert.NoError(t, err)
	chs.StdIn <- "ls -las /proc/version"
	// This will cause the channeler to run out of patience.
	time.Sleep(2 * p.ChTimeoutIn)
	chs.StdIn <- "exit 77"
	close(chs.StdIn)
	go consumeChannel("err", chs.StdErr)
	go consumeChannel("out", chs.StdOut)
	if err = <-chs.Done; assert.Error(t, err) {
		assert.Contains(
			t, err.Error(),
			"timeout of 50ms elapsed awaiting for input or close on stdin")
	}
}

func TestStartWithBackPressure(t *testing.T) {
	p := &Params{
		Path: theShell,
		// Use small buffer to create backpressure.
		BuffSizeOut:  1,
		ChTimeoutOut: 50 * time.Millisecond,
	}
	chs, err := Start(p)
	assert.NoError(t, err)
	chs.StdIn <- "ls -las /proc/version"
	chs.StdIn <- "ls -las /proc/version"
	chs.StdIn <- "exit 0"
	close(chs.StdIn)
	go consumeChannel("err", chs.StdErr)
	// Use a slow consumer to create backpressure.
	go func() {
		time.Sleep(2 * p.ChTimeoutOut)
		for line := range chs.StdOut {
			time.Sleep(2 * p.ChTimeoutOut)
			fmt.Printf("%s: %q\n", "out", line)
		}
	}()
	if err = <-chs.Done; assert.Error(t, err) {
		assert.Contains(
			t, err.Error(),
			"timeout of 50ms elapsed awaiting write to stdOut")
	}
}
