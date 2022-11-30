package shexec

import (
	"fmt"
	"time"
)

// execStateOff implements the "off" state of the Shell.
type execStateOff struct {
	infra *execInfra
}

func (exOff *execStateOff) subStart(d time.Duration) (execState, error) {
	if err := exOff.infra.infraStart(d); err != nil {
		return exOff, err
	}
	return &execStateIdle{infra: exOff.infra}, nil
}

func (exOff *execStateOff) subRun(_ time.Duration, _ Commander) (
	execState, error) {
	return exOff, fmt.Errorf("run called, but shell not started yet")
}

func (exOff *execStateOff) subStop(
	_ time.Duration, _ bareCommand) (execState, error) {
	return exOff, fmt.Errorf("stop called, but shell not started yet")
}
