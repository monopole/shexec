package scripter

import "time"

type bareCommand string

// execState is the internal representation of Executor state.
// Every executor state must implement execState.
type execState interface {
	subStart(time.Duration) (execState, error)
	subRun(time.Duration, Commander) (execState, error)
	subStop(time.Duration, bareCommand) (execState, error)
}
