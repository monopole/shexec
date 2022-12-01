package shexec_test

import (
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
