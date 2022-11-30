package shexec

import (
	"sync"
	"time"
)

// execMutex implements Shell.
// It allows for safe use of a Shell in a CSP environment.
// Instead of having a state variable and branching to distinguish states,
// execMutex allows each state to have a distinct implementation.
// The states share common code and infrastructure via execInfra.
type execMutex struct {
	state execState
	mutex sync.Mutex
}

func (r *execMutex) Start(d time.Duration) (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.state, err = r.state.subStart(d)
	return
}

func (r *execMutex) Run(d time.Duration, c Commander) (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.state, err = r.state.subRun(d, c)
	return
}

func (r *execMutex) Stop(d time.Duration, c string) (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.state, err = r.state.subStop(d, bareCommand(c))
	return
}
