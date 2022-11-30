package shexec

import (
	"fmt"
	"time"
)

// execStateIdle implements the "idle" state of the Shell.
type execStateIdle struct {
	infra *execInfra
}

func (exIdle *execStateIdle) subStart(_ time.Duration) (execState, error) {
	return exIdle, fmt.Errorf("start called, but shell is already started")
}

func (exIdle *execStateIdle) subRun(
	d time.Duration, c Commander) (execState, error) {
	if err := exIdle.infra.infraRun(d, c); err != nil {
		return &execStateOff{infra: exIdle.infra}, err
	}
	return exIdle, nil
}

func (exIdle *execStateIdle) subStop(
	d time.Duration, c bareCommand) (execState, error) {
	return &execStateOff{infra: exIdle.infra}, exIdle.infra.infraStop(d, c)
}
