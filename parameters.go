package shexec

import (
	"fmt"
	"github.com/monopole/shexec/channeler"
)

// Parameters is a bag of parameters for a Shell instance.
// See individual fields for their explanation.
type Parameters struct {
	channeler.Params

	// SentinelOut holds the command sent to the shell after every
	// command other than the exit command.
	// SentinelOut is used to be sure that output generated in the
	// course of running command N is swept up and accounted for
	// before looking for output from command N+1.
	SentinelOut Sentinel

	// SentinelErr is a command that intentionally triggers output
	// on stderr, e.g. a misspelled command, a command with a non-extant
	// flag - something that doesn't cause any real trouble.  If non-empty,
	// this is issued after every command other than the exit command,
	// either before or after issuing the OutSentinel command.
	// SentinelErr is used to be sure that any errors generated in the
	// course of running command N are swept up and accounted for before
	// looking for errors from command N+1.
	SentinelErr Sentinel
}

// Validate returns an error if there's a problem in the Parameters.
func (p *Parameters) Validate() error {
	if err := p.Params.Validate(); err != nil {
		return err
	}
	if err := p.SentinelOut.Validate(); err != nil {
		return fmt.Errorf("problem in SentinelOut; %w", err)
	}
	if p.SentinelErr.C != "" {
		if err := p.SentinelErr.Validate(); err != nil {
			return fmt.Errorf("problem in SentinelErr; %w", err)
		}
	}
	return nil
}
